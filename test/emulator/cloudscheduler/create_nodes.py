import os
import argparse
import json
from urllib.request import urlopen

def download_nodes(url) -> dict:
    with urlopen(url) as f:
        return json.loads(f.read())


def load_nodes(filename) -> dict:
    with open(filename, "r") as file:
        return json.load(file)


def build_tags(spec: dict) -> list:
    tags = [
        spec.get("node_type", ""),
        spec.get("project", ""),
        "bme680 microphone" if spec.get("shield", False) == True else "",
        "camera" if any([
            spec.get("top_camera", "none") != "none",
            spec.get("bottom_camera", "none") != "none",
            spec.get("left_camera", "none") != "none",
            spec.get("right_camera", "none") != "none"]) else ""
    ]
    return " ".join(tags).split()


def build_devices(spec: dict) -> list:
    devices = []
    node_type = spec.get("node_type", None)
    if node_type == None:
        print("Error: failed to find node type")
        return devices
    if node_type.lower() == "wsn":
        devices.append({
            "name": "ws-core",
            "architecture": "arm64",
            "resource": {
                "cpu": "6000m",
                "memory": "8Gi",
                "gpu_memory": "8Gi",
            }
        })
    elif node_type.lower() == "blade":
        devices.append({
            "name": "sb-core",
            "architecture": "amd64",
            "resource": {
                "cpu": "160000m",
                "memory": "32Gi",
                "gpu_memory": "16Gi"
            }
        })
    else:
        print(f'Error: node type ({node_type}) is unknown')
    if spec.get("shield", False) == True:
        devices.append({
            "name": "ws-rpi",
            "architecture": "arm64",
            "resource": {
                "cpu": "4000m",
                "memory": "4Gi",
            }
        })
    return devices


def build_hardware(spec):
    # NOTE: all Sage nodes have a GPU
    return {
        "gpu": True,
        "camera": True if any([
            spec.get("top_camera", "none") != "none",
            spec.get("bottom_camera", "none") != "none",
            spec.get("left_camera", "none") != "none",
            spec.get("right_camera", "none") != "none"]) else False
    }


def new_node(spec: dict) -> dict:
    return {
        "name": spec.get("vsn", ""),
        "tags": build_tags(spec),
        "devices": build_devices(spec),
        "hardware": build_hardware(spec),
    }


def save_node(path: str, node: dict):
    with open(os.path.join(path, f'{node.get("name")}.json'), "w") as file:
        file.write(json.dumps(node, indent=4))


def create_nodes(args):
    if args.node_url != "":
        nodes_raw = download_nodes(args.node_url)
    elif args.node_file != "":
        nodes_raw = load_nodes(args.node_file)
    else:
        print("Error: no node source is specified.")
        return 1
    for spec in nodes_raw:
        if spec.get("node_id", "") == "":
            continue
        vsn = spec.get("vsn", "none")
        if vsn == "none":
            continue
        print(f'Creating node for {vsn}...')
        node = new_node(spec)
        save_node(args.out_path, node)
        print("Done")
    return 0


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-node-url",
        dest="node_url",
        action="store",
        default="",
        type=str,
        help="URL path to list of nodes"
    )
    parser.add_argument(
        "-node-file",
        dest="node_file",
        action="store",
        default="",
        type=str,
        help="File path to list of nodes"
    )
    parser.add_argument(
        "-output-path",
        dest="out_path",
        action="store",
        required=True,
        type=str,
        help="Path to store node files"
    )
    return_code = create_nodes(parser.parse_args())
    if return_code != 0:
        print(parser.print_help())
    exit(return_code)