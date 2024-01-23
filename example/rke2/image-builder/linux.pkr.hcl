variable "omi_name" {
    type = string
    default = "${env("OMI_NAME")}"
}

variable "omi" {
    type = string
    default = "${env("SOURCE_OMI")}"
}

variable "volsize" {
    type = string
    default = "10"
}

variable "username" {
    type = string
    default = "outscale"
}

variable "script" {
    type = string
    default = "${env("SCRIPT_BASE")}"
}

variable "rke2_semver" {
    type = string
    default = "${env("RKE2_SEMVER")}"
}

variable "region" {
    type = string
    default = "${env("OUTSCALE_REGION")}"
}

source "outscale-bsu" "builder" {
    force_deregister = true
    force_delete_snapshot = true
    omi_account_ids = ["040667503696"]
    omi_name = "${var.omi_name}"
    source_omi = "${var.omi}"
    ssh_interface = "public_ip"
    ssh_username = "${var.username}"
    vm_type = "tinav6.c2r4p1"
}

build {
    sources = [ "source.outscale-bsu.builder" ]
    provisioner "file" {
        source = "./files/06_hotplug.cfg"
        destination = "/tmp/06_hotplug.cfg"
    }
    provisioner "shell" {
        execute_command = "chmod +x {{ .Path }}; {{ .Vars }} sudo -E -S bash -x '{{ .Path }}' '${var.rke2_semver}'"
        scripts = [
            "./script/bootstrap.sh",
        ]
    }
}
