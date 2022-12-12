use crate::core::{self, Datas};
use lazy_static::lazy_static;
use regex::Regex;
use serde::{Deserialize, Serialize};
use std::error;
use std::process::exit;

#[derive(Deserialize, Serialize, Debug, Clone)]
pub struct KubernetesVersions {
    pub kubernetes_semver: String,
    pub kubernetes_series: String,
}

#[derive(Deserialize, Serialize, Debug, Clone)]
pub struct KubernetesSpecs {
    pub build_timestamp: Option<String>,
    pub kubernetes_deb_version: Option<String>,
    pub kubernetes_rpm_version: Option<String>,
    pub kubernetes_semver: Option<String>,
    pub kubernetes_series: Option<String>,
}

#[derive(Deserialize, Serialize, Debug, Clone)]
#[serde(rename_all = "camelCase")]
pub struct Input {
    pub kubernetes: Vec<KubernetesVersions>,
}
// from will turn input into data
impl From<Input> for core::Datas {
    fn from(input: Input) -> Self {
        let mut datas = core::Datas { datas: Vec::new() };
        input.fill_data_kubernetes(&mut datas);
        datas
    }
}

impl Input {
    // new will create vector of kubernetes version based on params
    pub fn new(
        kubernetes_semver: &str,
        kubernetes_series: &str,
        format: &str,
    ) -> Result<Input, Box<dyn error::Error>> {
        let data = match format {
            "params" => {
                let mut kubernetes = Vec::<KubernetesVersions>::new();
                let kubernetes_version = KubernetesVersions {
                    kubernetes_semver: kubernetes_semver.to_string(),
                    kubernetes_series: kubernetes_series.to_string(),
                };
                kubernetes.push(kubernetes_version);
                Input { kubernetes }
            }
            _ => {
                println!("Unknown format {:#?}", format);
                exit(1);
            }
        };
        Ok(data)
    }
    // fill_data_kubernetes add in data kubernetes spec for image-builder
    fn fill_data_kubernetes(&self, datas: &mut Datas) {
        for kubernetes_version in &self.kubernetes {
            let specs = match KubernetesSpecs::new(kubernetes_version) {
                Some(specs) => specs,
                None => continue,
            };
            let core_kubernetes = core::KubernetesImage {
                kubernetes_semver: specs.kubernetes_semver,
                kubernetes_series: specs.kubernetes_series,
                kubernetes_deb_version: specs.kubernetes_deb_version,
                kubernetes_rpm_version: specs.kubernetes_rpm_version,
                build_timestamp: specs.build_timestamp,
            };
            datas
                .datas
                .push(core::Data::KubernetesImage(core_kubernetes))
        }
    }
}
impl KubernetesSpecs {
    // new will create kubernetesSpec for imageBuilder based on kubernetes params
    fn new(kubernetes: &KubernetesVersions) -> Option<Self> {
        let kubernetes_semver = &kubernetes.kubernetes_semver;
        let kubernetes_series = &kubernetes.kubernetes_series;
        lazy_static! {
            static ref REG_KUBERNETES_SEMVER: Regex =
                Regex::new(r"^v\d{1}.\d{2}.\d{1,2}$").unwrap();
            static ref REG_KUBERNETES_SERIES: Regex = Regex::new(r"^v\d{1}.\d{2}$").unwrap();
        }
        match REG_KUBERNETES_SEMVER.is_match(kubernetes_semver.as_str()) {
            true => println!("{} has good format", kubernetes_semver),
            false => {
                println!("{} has bad format", kubernetes_semver);
                exit(1);
            }
        }
        match REG_KUBERNETES_SERIES.is_match(kubernetes_series.as_str()) {
            true => println!("{} has good format", kubernetes_series),
            false => {
                println!("{} has bad format", kubernetes_series);
                exit(1);
            }
        }
        let out = KubernetesSpecs {
            kubernetes_semver: Some(String::from(kubernetes_semver)),
            kubernetes_series: Some(String::from(kubernetes_series)),
            kubernetes_deb_version: None,
            kubernetes_rpm_version: None,
            build_timestamp: None,
        };
        Some(out)
    }
}
