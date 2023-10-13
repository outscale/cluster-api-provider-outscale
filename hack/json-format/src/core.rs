use chrono::prelude::*;
use log::warn;
use serde::{Deserialize, Serialize};
use std::error::Error;
use std::fmt;

#[derive(Serialize, Deserialize, Debug)]
#[serde(untagged)]
pub enum Data {
    KubernetesImage(KubernetesImage),
}

pub struct Datas {
    pub datas: Vec<Data>,
}

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct KubernetesImage {
    pub build_timestamp: Option<String>,
    pub kubernetes_deb_version: Option<String>,
    pub kubernetes_rpm_version: Option<String>,
    pub kubernetes_semver: Option<String>,
    pub kubernetes_series: Option<String>,
}

impl DataTrait for KubernetesImage {
    // set_format set kubernetes spec based on kubernetes version spec
    fn set_format(&mut self) -> Result<(), DataError> {
        let time_now = Utc::now();
        let timestamp = time_now.timestamp();
        self.build_timestamp = Some(timestamp.to_string());
        let kubernetes_version = String::from(
            self.kubernetes_semver
                .to_owned()
                .as_deref()
                .unwrap_or("v1.22.1"),
        );
        let kubernetes_deb_version = String::from(&kubernetes_version.replace("v", "")) + "-1.1";
        self.kubernetes_deb_version = Some(kubernetes_deb_version);
        let kubernetes_rpm_version = String::from(&kubernetes_version.replace("v", ""));
        self.kubernetes_rpm_version = Some(kubernetes_rpm_version);
        println!(
            "kubernetes json build timestamp: {}",
            String::from(
                self.build_timestamp
                    .to_owned()
                    .as_deref()
                    .unwrap_or("nightly")
            )
        );
        println!(
            "kubernetes semvers: {}",
            self.kubernetes_semver
                .to_owned()
                .as_deref()
                .unwrap_or("v1.22.1")
        );
        println!(
            "kubernetes series: {}",
            self.kubernetes_series
                .to_owned()
                .as_deref()
                .unwrap_or("v1.22")
        );
        println!(
            "kubernetes rpm version: {:?}",
            self.kubernetes_rpm_version
                .to_owned()
                .as_deref()
                .unwrap_or("1.22.1-1.1")
        );
        println!(
            "kubernetes deb version: {:?}",
            self.kubernetes_deb_version
                .to_owned()
                .as_deref()
                .unwrap_or("1.22.1")
        );
        Ok(())
    }
}

#[derive(Serialize, Deserialize, Debug)]
pub struct DataError {
    pub output: String,
}

impl fmt::Display for DataError {
    fn fmt(&self, f: &mut fmt::Formatter) -> fmt::Result {
        write!(f, "{}", self.output)
    }
}
impl Error for DataError {
    fn description(&self) -> &str {
        &self.output
    }
}
impl Datas {
    // set_format set format for kubernetes image
    pub fn set_format(&mut self) -> Result<(), DataError> {
        for data in self.datas.iter_mut() {
            match data {
                Data::KubernetesImage(kubernetes_image) => kubernetes_image.set_format()?,
            }
        }
        Ok(())
    }
    // json set data to json format with string
    pub fn json(&self) -> serde_json::Result<String> {
        let mut out = String::new();
        for data in &self.datas {
            match serde_json::to_string(data) {
                Ok(serialized) => out.push_str(serialized.as_str()),
                Err(e) => {
                    warn!("trouble to serialize json: {}", e);
                    continue;
                }
            }
            out.push('\n');
        }
        out.pop();
        Ok(out)
    }
}

trait DataTrait {
    fn set_format(&mut self) -> Result<(), DataError>;
}

#[cfg(test)]
mod datas_test {
    use super::*;

    #[test]
    // set_format_test validate set_format function
    fn set_format_test() {
        let kubernetes_image = KubernetesImage {
            build_timestamp: Some("".to_string()),
            kubernetes_deb_version: Some("".to_string()),
            kubernetes_rpm_version: Some("".to_string()),
            kubernetes_semver: Some("v1.22.1".to_string()),
            kubernetes_series: Some("v1.22".to_string()),
        };
        let data = Data::KubernetesImage(kubernetes_image);
        let mut datas = Vec::<Data>::new();
        datas.push(data);
        let mut data_struct = Datas { datas: datas };
        if let Err(error) = data_struct.set_format() {
            warn!("can not set format {}", error);
        }
        for kubernetes_datas in data_struct.datas {
            let Data::KubernetesImage(kubernetes) = kubernetes_datas;
            println!("value: {:#?}", kubernetes);
            assert_eq!(
                kubernetes.kubernetes_rpm_version,
                Some("1.22.1".to_string())
            )
        }
    }

    #[test]
    // json_test validate json function
    fn json_test() {
        let kubernetes_image = KubernetesImage {
            build_timestamp: Some("1670519118".to_string()),
            kubernetes_deb_version: Some("1.22.1-00".to_string()),
            kubernetes_rpm_version: Some("1.22.1-0".to_string()),
            kubernetes_semver: Some("v1.22.1".to_string()),
            kubernetes_series: Some("v1.22".to_string()),
        };
        let data = Data::KubernetesImage(kubernetes_image);

        let mut datas = Vec::<Data>::new();
        datas.push(data);
        let data_struct = Datas { datas: datas };
        let output = match data_struct.json() {
            Ok(json_format) => json_format,
            Err(error) => error.to_string(),
        };
        assert_eq!(output, "{\"build_timestamp\":\"1670519118\",\"kubernetes_deb_version\":\"1.22.1-00\",\"kubernetes_rpm_version\":\"1.22.1-0\",\"kubernetes_semver\":\"v1.22.1\",\"kubernetes_series\":\"v1.22\"}");
    }
}
