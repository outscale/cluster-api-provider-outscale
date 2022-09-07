trigger_mode(TRIGGER_MODE_MANUAL)

docker_build(os.getenv("CONTROLLER_IMAGE", ""), ".")

allow_k8s_contexts(os.getenv("K8S_CONTEXT", "phandalin"))

k8s_yaml("capm.yaml")
