use clap::Parser;
use log::error;
use serde::{Deserialize, Serialize};
use std::error;
use std::process::exit;
mod core;
mod input;

#[derive(Serialize, Deserialize, Debug, Clone)]
pub struct KubernetesVersion {
    pub kubernetes_output: String,
}
// create_kubernete_json create json file for image-builder based on kubernetes version parameters
pub fn create_kubernetes_json() -> KubernetesVersion {
    #[derive(Parser, Debug)]
    #[command(author, version, about, long_about=None)]
    struct Args {
        #[arg(long, default_value_t = String::from("params"))]
        format: String,
        #[arg(long, short = 'i')]
        input: Option<String>,
        #[arg(long, short = 'o')]
        output: Option<String>,
        #[arg(long, short = 'r')]
        kubernetes_semver: Option<String>,
        #[arg(long, short = 'v')]
        kubernetes_series: Option<String>,
    }
    let args = Args::parse();
    let mut datas: core::Datas;
    let input = match args.format.as_str() {
        "params" => {
            let kubernetes_default_semver = "v1.22.1";
            let kubernetes_semver = match args.kubernetes_semver.as_deref() {
                Some(kubernetes_semver) => kubernetes_semver,
                None => kubernetes_default_semver,
            };
            let kubernetes_default_series = "v1.22";
            let kubernetes_series = match args.kubernetes_series.as_deref() {
                Some(kubernetes_series) => kubernetes_series,
                None => kubernetes_default_series,
            };

            match input::Input::new(kubernetes_semver, kubernetes_series, "params") {
                Ok(kubernetes_versions) => kubernetes_versions,
                Err(err) => {
                    println!("{:#?}", err);
                    exit(1);
                }
            }
        }
        _ => {
            println!("Unknown format {:#?}", args.input);
            exit(1);
        }
    };

    datas = core::Datas::from(input);
    if let Err(error) = datas.set_format() {
        error!("can not set format {}", error);
        exit(1);
    }

    let output: String = match datas.json() {
        Ok(json_format) => json_format,
        Err(error) => {
            error!("{}", error);
            exit(1);
        }
    };
    if let Some(output_file) = args.output.as_deref() {
        write_to_json(output_file, output).unwrap_or_else(|error| {
            error!("Can not create json file {:?}", error);
            exit(1);
        });
    } else {
        println!("{}", output);
    }
    KubernetesVersion {
        kubernetes_output: match datas.json() {
            Ok(json_format) => json_format,
            Err(error) => {
                error!("{}", error);
                exit(1);
            }
        },
    }
}

pub fn main() {
    create_kubernetes_json();
}
// write_to_json write json file
pub fn write_to_json(output_path: &str, output_data: String) -> Result<(), Box<dyn error::Error>> {
    let output_json_data: core::Data = serde_json::from_str(output_data.as_str()).unwrap();
    match std::fs::write(
        output_path,
        serde_json::to_string_pretty(&output_json_data).unwrap(),
    ) {
        Ok(file) => Ok(file),
        Err(err) => {
            error!("Can not create json file {}", err);
            exit(1);
        }
    }
}
