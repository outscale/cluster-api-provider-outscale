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

import argparse
import re
from datetime import datetime, timedelta
import os
from osc_sdk_python import Gateway as OSCGateway


# ParseArgs set parameters
def parseArgs():
    """Read Cli args"""
    parser = argparse.ArgumentParser(description="Parse command line args")
    parser.add_argument('--days', action='store', default=30, type=int, help='Clean up omi older than X days (default = 30 days)'),
    parser.add_argument('--owner', action='store', default='123456789123', type=str, help='Outscale account owner ID (fictional default account number)'),
    parser.add_argument('--imageId', action='store', type=str, help='Outscale Image Id',)
    parser.add_argument('--imageName', action='store', type=str, help='Outscale Image Name',)
    parser.add_argument('--imageNameFilterPath', action='store', type=str, help='file path of Outscale Image Name to keep',)  
    parser.add_argument('--imageNamePattern', action='store', type=str, help='Outscale Image Name Pattern',)
    return parser.parse_args()

# checkImageIdAttached check imageId exist
def checkImageIdAttached(gateway, imageId, accountOwnerImageId):
    """Check Image Id"""
    vms = gateway.ReadVms().get('Vms', [])
    vmsImageId = [vm["ImageId"] for vm in vms if 'ReservationId' in vm]
    for vmImageId in vmsImageId:
        if vmImageId == imageId:
            print("Get Vm Image {}".format(imageId))
            return True
    return False

# deleteImageId delete imageId
def deleteImageId(gateway, imageId, accountOwnerImageId):
    """Delete Image Id """
    res = gateway.DeleteImage(ImageId=imageId)
    if "Errors" in res:
        print("Can not delete Image {} because {}".format(imageId, res))
    print("Delete ImageId {}".format(imageId))

# getImageId retrieve imageId        
def getImageId(gateway, imageName, accountOwnerImageId):
    """Get Image Id"""
    for image in gateway.ReadImages(Filters={'ImageNames': [imageName], 'AccountIds': [accountOwnerImageId]}).get('Images', []):
        imageId = image['ImageId']
        print("Get ImageId {}".format(imageId))
        return imageId

# checkImageIdExist check imageId exist
def checkImageIdExist(gateway, imageId, accountOwnerImageId):
    """Check Image Id exist"""
    for image in gateway.ReadImages(Filters={'AccountIds': [accountOwnerImageId]}).get('Images', []):
        if image['ImageId'] == imageId:
            print("ImageId {} does exist".format(imageId))
            return True
    print("ImageId {} does not exist".format(imageId))
    return False
    
# deleteImage delete image.
def deleteImage(gateway, accountOwnerImageId, daysCount, imageNamePattern, imageNameFilter):
    """Get Image Date"""
    for image in gateway.ReadImages(Filters={'AccountIds': [accountOwnerImageId]}).get('Images', []):
        imageId = image['ImageId']
        creationDate = image['CreationDate']
        imageName = image['ImageName']
        match = re.search(imageNamePattern, imageName)
        if imageNamePattern != "" and match and imageName not in imageNameFilter:
            print("Image {} match imagePattern {}".format(imageName, imageNamePattern))
            print("Get ImageName {} with ImageId {} in {}".format(imageName, imageId, creationDate))
            checkImageToBeDelete(gateway, accountOwnerImageId, creationDate, imageId, daysCount)
        else:    
            print("ImageName {} will be keeped".format(imageName)) 


# checkImageToBeDelete check if image need to be delete
def checkImageToBeDelete(gateway, accountOwnerImageId, creationDate, imageId, daysCount):
    """Check Image DateTime"""
    omiAgeLimit = datetime.now() - timedelta(days=daysCount)
    imageDate = datetime.strptime(creationDate, '%Y-%m-%dT%H:%M:%S.%fZ')
    if imageDate < omiAgeLimit:
        print("ImageId {} will be deleted because its is more than {} days".format(imageId, daysCount))
        deleteImageId(gateway, imageId, accountOwnerImageId)
    else:
        print("ImageId {} will be saved because its is less than {} days".format(imageId, daysCount))

def readImageNameFilter(file_path) -> list:
    """Read ImageName to keep file"""
    filtered_imagename_list = list()
    with open(file_path, 'r') as file:
        for line in file:
            if not line.strip().startswith("ubuntu") and not line.strip().startswith("centos"):
    	        continue
            if line.strip() in filtered_imagename_list:
                continue
            filtered_imagename_list.append(line.strip())
    return filtered_imagename_list
    
    
def main():
    """Main function to delete old image
    """
    access_key = os.environ['OSC_ACCESS_KEY']
    secret_key = os.environ['OSC_SECRET_KEY']
    region = os.environ['OSC_REGION']

    args = parseArgs()
    daysCount = args.days
    imageId = args.imageId
    accountOwnerImageId = args.owner
    imageName = args.imageName
    imageNamePattern = args.imageNamePattern
    imageNameFilterPath = args.imageNameFilterPath
    imageNameFilter = readImageNameFilter(imageNameFilterPath)
    print("Cleanup image")
    gateway = OSCGateway(
        **{"access_key": access_key, "secret_key": secret_key, "region": region}
    )
    if imageName != None:
        imageId = getImageId(gateway, imageName, accountOwnerImageId)
    if imageId != None:
        print("Image {} with id {} will be checked".format(imageName, imageId))
        checkImageId = checkImageIdExist(gateway, imageId, accountOwnerImageId)
        if checkImageId:
	        deleteImageId(gateway, imageId, accountOwnerImageId)
    else:
        deleteImage(gateway, accountOwnerImageId, daysCount, imageNamePattern, imageNameFilter)


if __name__ == "__main__":
    main()
