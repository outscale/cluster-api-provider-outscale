
docker_build(os.getenv('CONTROLLER_IMAGE', '042b4721a38342028d65c28be2b30e64-157001637.eu-west-2.lbu.outscale.com:5000/controller'), '.')


allow_k8s_contexts(os.getenv('K8S_CONTEXT', 'phandalin'))

k8s_yaml('capm.yaml')
