import os
import re
import json
import argparse

from urllib.request import urlopen

default_nx = {
    "name": "nxcore",
    "architecture": "arm64",
    "resource": {
        "cpu": "6000m",
        "memory": "8Gi",
        "gpu_memory": "8Gi"
    },
    "hardware": ["gpu"]
}

default_nxagent = {
    "name": "nxagent",
    "architecture": "arm64",
    "resource": {
        "cpu": "6000m",
        "memory": "8Gi",
        "gpu_memory": "8Gi"
    },
    "hardware": ["gpu"]
}

default_pi = {
    "name": "rpi",
    "architecture": "arm64",
    "resource": {
        "cpu": "4000m",
        "memory": "4Gi",
    },
}

default_dell = {
    "name": "dell",
    "architecture": "amd64",
    "resource": {
        "cpu": "16000m",
        "memory": "32Gi",
        "gpu_memory": "16Gi"
    },
    "hardware": ["gpu"]
}

def query_node_monitoring_sheet(url):
    with urlopen(url) as f:
        return json.loads(f.read())


def main(args):
    nodes = query_node_monitoring_sheet(args.url)
    for node in nodes:
        if not re.match("^[0-9a-zA-Z]{16}$", node["node_id"]):
            continue
        output = {
            "name": node["vsn"],
            "hardware": {},
            "ontology": {},
        }
        devices = []
        if node["node_type"].lower() == "wsn":
            devices.append(default_nx)
            output["hardware"]["bme280"] = True
        elif node["node_type"].lower() == "blade":
            devices.append(default_dell)

        if node["nx_agent"] == True:
            devices.append(default_nxagent)
        
        if node["shield"] == True:
            devices.append(default_pi)
            output["hardware"]["bme680"] = True
            output["hardware"]["microphone"] = True
        
        if node["top_camera"] != "none":
            output["hardware"]["top_camera"] = node["top_camera"]
            output["hardware"]["camera"] = True
        if node["bottom_camera"] != "none":
            output["hardware"]["bottom_camera"] = node["bottom_camera"]
            output["hardware"]["camera"] = True
        if node["left_camera"] != "none":
            output["hardware"]["left_camera"] = node["left_camera"]
            output["hardware"]["camera"] = True
        if node["right_camera"] != "none":
            output["hardware"]["right_camera"] = node["right_camera"]
            output["hardware"]["camera"] = True
        
        output["ontology"]["gps_lat"] = node["gps_lat"]
        output["ontology"]["gps_lon"] = node["gps_lon"]
        output["devices"] = devices
        with open(os.path.join(args.base_path, output["name"]+".json"), "w") as file:
            file.write(json.dumps(output, indent=4))

if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument(
        '-base-path', dest='base_path',
        action='store', default = 'data/nodes', type=str,
        help='Base path attached to the node IDs')
    parser.add_argument(
        '-url', dest='url',
        action='store', type=str, required=True,
        help='URL of the node management')
    main(parser.parse_args())
