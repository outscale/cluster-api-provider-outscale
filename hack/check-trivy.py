#!/usr/bin/env python3

# Copyright 2022 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from typing import Dict
import requests
import json
import logging
import argparse
import sys

ALPINE_URL = "https://secdb.alpinelinux.org"
ALPINE_JSON = "community.json"
ALPINE_UNFIX_URL = "https://raw.githubusercontent.com/aquasecurity/vuln-list/main/alpine-unfixed/"
TRIVY_IGNORE_PATH = ".trivyignore"
DISTRIBUTION_TYPE = "community"
AWS_S3_UNFIXED_CVE_URL = "https://raw.githubusercontent.com/aquasecurity/vuln-list/main/glad/go/github.com/aws/aws-sdk-go/service/s3/s3crypto/"
AWS_S3_UNFIXED_GHSA_URL = "https://raw.githubusercontent.com/aquasecurity/vuln-list/main/ghsa/go/github.com/aws/aws-sdk-go/service/s3/s3crypto/"
def retrieve_alpine_data(distribution) -> Dict:
    """Retrieve alpine cve"""
    resp = requests.get(ALPINE_URL + "/v" + distribution + "/" + ALPINE_JSON)
    if resp.status_code != 200:
        print("Error while retrieving the data from {}".format(ALPINE_URL))
        return None
    return json.loads(resp.content)

def retrieve_unfixed_cve(cve_version, distribution, logger) -> bool:
    """Check if cve is unfixed or does not exist"""
    resp = requests.get(ALPINE_UNFIX_URL + cve_version + ".json")
    if resp.status_code != 200:
       logger.warning("The cve {} has not been found in alpine {}".format(cve_version, distribution))
       return False
    unfixed_cve = json.loads(resp.content)
    length = len(unfixed_cve["state"])
    distribution_version = distribution + "-" + DISTRIBUTION_TYPE
    for i in range(length):
        state = unfixed_cve["state"][i]
        if state["repo"] == distribution_version:
            logger.info("the latest version {} of {} in {} has not yet fixed the cve {}".format(state["packageVersion"], state["packageName"], state["repo"], cve_version))
    return True

def retrieve_unfixed_aws_s3_cve(cve_version, distribution, logger) -> bool:
    """Check aws s3 cve"""
    if cve_version.startswith("CVE"):
        resp = requests.get(AWS_S3_UNFIXED_CVE_URL + cve_version + ".json")
        if resp.status_code != 200:
            logger.warning("The cve {} has not been found in aws s3".format(cve_version))
            return False
    elif cve_version.startswith("GHSA"):
        resp = requests.get(AWS_S3_UNFIXED_GHSA_URL + cve_version + ".json")
        if resp.status_code != 200:
            logger.warning("the cve {} has not been found".format(cve_version))
            return False
    return True

def read_trivy_filter(file_path) -> list:
    """Read trivy ignore file"""
    filtered_cve_list = list()
    with open(file_path, 'r') as file:
        for line in file:
            if not line.strip().startswith("CVE") and not line.strip().startswith("GHSA"):
                continue
            if line.strip() in filtered_cve_list:
                continue
            filtered_cve_list.append(line.strip())
    return filtered_cve_list

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument("--trivy-ignore", type=str, required=True, help="trivyignore file path")
    parser.add_argument("--distribution", type=str, required=True, help="alpine version with major, minor and without patch (ex: 3.16)")
    args = parser.parse_args()
    logger = logging.getLogger("check_trivy")
    logger.setLevel(logging.INFO)
    ch = logging.StreamHandler()
    ch.setLevel(logging.DEBUG)
    formatter = logging.Formatter('%(asctime)s - %(name)s - %(levelname)s - %(message)s')
    ch.setFormatter(formatter)
    logger.addHandler(ch)
    all_cves = retrieve_alpine_data(args.distribution)
    if all_cves is None:
        sys.exit(1)
    resolved_cve = False
    ignored_cves = read_trivy_filter(args.trivy_ignore)
    for package in all_cves.keys():
        if package == "packages":
           length = len(all_cves[package])
           for i in range(length):
               package_pkg = all_cves[package][i]['pkg']
               package_pkg_secfixes = package_pkg['secfixes']
               for pkg_version in package_pkg_secfixes.keys():
                   for cve_version in enumerate(package_pkg_secfixes[pkg_version]):
                       if cve_version[1] in ignored_cves:
                           logger.warning("{} has resolved {} in release {}".format(package_pkg['name'], cve_version[1], pkg_version))
                           resolved_cve = True
                           ignored_cves.remove(cve_version[1])
    ignored_cves_list = []
    for ignored_cve in ignored_cves:     
        unfixed_cve = retrieve_unfixed_cve(ignored_cve, args.distribution, logger)
        unfixed_aws_s3_cve = retrieve_unfixed_aws_s3_cve(ignored_cve, args.distribution, logger)
        if unfixed_aws_s3_cve or unfixed_cve:
            ignored_cves_list.append(ignored_cve)
    ignored_cves_not_found = [x for x in ignored_cves if x not in ignored_cves_list]
    if resolved_cve:
        sys.exit(1)
    elif len(ignored_cves_not_found) != 0:
        logger.info("These CVE has not been found in alpine or s3 cve: {}".format(ignored_cves_not_found))
 
    
