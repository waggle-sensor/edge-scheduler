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


def build_required_hardware(spec):
    # TODO: We must figure out what hardware is required for plugin to run
    #       e.g., GPU, camera, microphone, etc
    required_hardware = ""
    # return " ".join(required_hardware).split()
    return {}


def build_arch(spec: dict) -> list:
    archs = [a.split("/")[1] for a in spec["source"]["architectures"]]
    return " ".join(archs).split()


def new_plugin(spec: dict) -> dict:
    return {
        "name": spec.get("name", ""),
        "image": spec.get("id", ""),
        "tags": spec.get("tags"),
        "required_hardware": build_required_hardware(spec),
        "architecture": build_arch(spec),
        "profile": [],
    }


def save(path: str, plugin: dict):
    with open(os.path.join(path, f'{plugin.get("image").replace("/", "-")}.json'), "w") as file:
        file.write(json.dumps(plugin, indent=4))


def create_plugins(args):
    if args.plugin_url != "":
        plugin_raw = download_nodes(args.plugin_url)
    elif args.plugin_file != "":
        plugin_raw = load_nodes(args.plugin_file)
    else:
        print("Error: no plugin source is specified.")
        return 1
    for plugin_spec in plugin_raw["data"]:
        if plugin_spec.get("id", "") == "":
            continue
        print(f'Creating plugin for {plugin_spec.get("id")}...')
        plugin = new_plugin(plugin_spec)
        save(args.out_path, plugin)
        print("Done")
    return 0


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "-plugin-url",
        dest="plugin_url",
        action="store",
        default="",
        type=str,
        help="URL path to list of plugins"
    )
    parser.add_argument(
        "-plugin-file",
        dest="plugin_file",
        action="store",
        default="",
        type=str,
        help="File path to list of plugins"
    )
    parser.add_argument(
        "-output-path",
        dest="out_path",
        action="store",
        required=True,
        type=str,
        help="Path to store plugin files"
    )
    return_code = create_plugins(parser.parse_args())
    if return_code != 0:
        print(parser.print_help())
    exit(return_code)