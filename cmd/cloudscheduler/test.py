from http import HTTPStatus
import requests
import pytest
import subprocess
from contextlib import ExitStack
from shutil import rmtree
from pathlib import Path

# SEAN
#
# The recommended way of using this test suite is simply to run:
# go build && pytest test.py
#
# This ensures that the program builds and that we're using the latest build for the test.
#
# A couple assumptions going into this. First, the cloudscheduler has a couple "fake" components with
# hardcoded data to allow for testing. Right now, here are the main items to be aware of:
#
#   1. Has "admintoken" token which authenticates as a superuser.
#   2. Has "usertoken" token which authenticates as a regular user.
#   3. All users have permissions on W01C and W022 and no other nodes.
#
# TODO(sean) Isolate server runs between tests!

# curl http://localhost:9770/api/v1/jobs/list


@pytest.fixture(autouse=True)
def wrap_tests():
    # isolate cloud scheduler data between runs
    data_dir = Path("data")
    rmtree(data_dir)
    data_dir.mkdir(parents=True)

    with ExitStack() as es:
        proc = es.enter_context(
            subprocess.Popen("./cloudscheduler", stdout=subprocess.PIPE)
        )
        es.callback(proc.terminate)
        for line in proc.stdout:
            if b"server starts" in line:
                break
        yield


def test_submit_requires_auth():
    r = requests.post(f"http://localhost:9770/api/v1/submit")
    assert r.status_code == HTTPStatus.UNAUTHORIZED
    data = r.json()
    assert data == {"error": "No token found"}


def test_submit_with_no_data():
    headers = {
        "Authorization": "Sage usertoken",
    }
    r = requests.post(f"http://localhost:9770/api/v1/submit", headers=headers)
    assert r.status_code == HTTPStatus.BAD_REQUEST
    data = r.json()
    assert data["error"] != ""
    assert "validation failed" in data["message"]


def test_submit_with_bad_data():
    headers = {
        "Authorization": "Sage usertoken",
    }
    r = requests.post(
        f"http://localhost:9770/api/v1/submit",
        headers=headers,
        data="some made up random data!",
    )
    assert r.status_code == HTTPStatus.BAD_REQUEST
    data = r.json()
    assert "yaml: unmarshal errors" in data["error"]


def test_submit_requires_node_permission():
    headers = {
        "Authorization": "Sage usertoken",
    }
    r = requests.post(
        f"http://localhost:9770/api/v1/submit",
        headers=headers,
        data="""name: dbaserh
plugins:
  - name: panda-dbaserh
    pluginSpec:
      image: registry.gitlab.com/lbl-anp/panda/dawn_pipeline/plugin-dbaserh:v1.0.9
      privileged: true
      selector:
        zone: core
      env:
        CRYSTAL_SER: 65008-00953
        DBASERH_FGN: "1.14"
        DBASERH_HV: "950"
        DBASERH_LLD: "12"
        DBASERH_SER: "0"
      resource:
        limit.cpu: 500m
        limit.memory: 500Mi
        request.cpu: 100m
        request.memory: 100Mi
nodeTags: []
nodes:
  W01A: null
scienceRules:
  - "schedule(panda-dbaserh): True"
successCriteria: []
""",
    )
    assert r.status_code == HTTPStatus.BAD_REQUEST


def test_submit_requires_plugin_in_ecr():
    headers = {
        "Authorization": "Sage usertoken",
    }
    r = requests.post(
        f"http://localhost:9770/api/v1/submit",
        headers=headers,
        data="""name: dbaserh
plugins:
  - name: panda-dbaserh
    pluginSpec:
      image: registry.gitlab.com/lbl-anp/panda/dawn_pipeline/plugin-dbaserh:v1.0.9
      privileged: true
      selector:
        zone: core
      env:
        CRYSTAL_SER: 65008-00953
        DBASERH_FGN: "1.14"
        DBASERH_HV: "950"
        DBASERH_LLD: "12"
        DBASERH_SER: "0"
      resource:
        limit.cpu: 500m
        limit.memory: 500Mi
        request.cpu: 100m
        request.memory: 100Mi
nodeTags: []
nodes:
  W01C: null
scienceRules:
  - "schedule(panda-dbaserh): True"
successCriteria: []
""",
    )
    assert r.status_code == HTTPStatus.BAD_REQUEST


def test_submit_success():
    headers = {
        "Authorization": "Sage usertoken",
    }

    r = requests.post(
        f"http://localhost:9770/api/v1/submit",
        headers=headers,
        data="""name: dbaserh
plugins:
  - name: cloud-motion
    pluginSpec:
      image: registry.sagecontinuum.org/bhupendraraut/cloud-motion:0.22.11.10
nodeTags: []
nodes:
  W01C: null
  W022: null
scienceRules:
  - "schedule(panda-dbaserh): True"
successCriteria: []
""",
    )
    # TODO(sean) Use ACCEPTED instead of OK?
    assert r.status_code == HTTPStatus.OK

    r = requests.get("http://localhost:9770/api/v1/jobs/list")
    r.raise_for_status()
    jobs = r.json()
    assert len(jobs) == 1
    # NOTE(sean) jobs currently returns a map of stringy ints -> jobs
    assert jobs["1"]["name"] == "dbaserh"
    assert jobs["1"]["user"] == "user"
    # TODO(sean) test more fields here!
