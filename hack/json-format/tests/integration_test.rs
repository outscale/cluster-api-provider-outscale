extern crate json_format;
use json_format::create_kubernetes_json;

#[test]
// success check create_kubernetes_json is a success
fn success() {
    let kubernetes_version = create_kubernetes_json();
    let mut kubernetes_image = kubernetes_version.kubernetes_output;
    let _ = kubernetes_image.drain(1..32);
    assert_eq!(kubernetes_image, "{\"kubernetes_deb_version\":\"1.22.1-1.1\",\"kubernetes_rpm_version\":\"1.22.1\",\"kubernetes_semver\":\"v1.22.1\",\"kubernetes_series\":\"v1.22\"}");
}
