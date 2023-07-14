import json
import urllib.request

import streamlit as st
import pandas as pd
from sage_data_client import query
import plotly.express as px

from utils import parse_events, fill_completion_failure

st.set_page_config(layout="wide")


@st.cache_data
def get_events(start, vsn=""):
    print("get_events called")
    if vsn == "":
        return query(
        start=start,
        filter={
            "name": "sys.scheduler.status.plugin.*",
            }
        )
    else:
        return query(
        start=start,
        filter={
            "name": "sys.scheduler.status.plugin.*",
            "vsn": vsn,
            }
        )


@st.cache_data
def get_production_vsn():
    url = "https://api.sagecontinuum.org/production"
    contents = urllib.request.urlopen(url).read().decode()
    production_nodes = json.loads(contents)
    return pd.DataFrame(production_nodes)


@st.cache_data
def get_all_jobs():
    url = f'https://es.sagecontinuum.org/api/v1/jobs/list'
    return json.loads(urllib.request.urlopen(url).read().decode())


@st.cache_data
def get_job_information(job_id):
    url = f'https://es.sagecontinuum.org/api/v1/jobs/{job_id}/status'
    contents = urllib.request.urlopen(url).read().decode()
    return json.loads(contents)


@st.cache_data
def get_manifest():
    url = "https://auth.sagecontinuum.org/manifests/"
    return json.loads(urllib.request.urlopen(url).read().decode())


def _iter_plugins(jobs):
    for _, j in jobs.iterrows():
        print(j)
        for p in j.plugins:
            yield j["name"], p["name"]


def check_node(node_jobs, node_data):
    errors = []
    if len(node_data) == 0:
        errors.append("node has no data to show")
        return errors
    vsn = node_data.vsn.unique()[0]
    # check 1: are the plugins in the active jobs being scheduled on the node
    for job_name, plugin_name in _iter_plugins(node_jobs):
        if node_data["k3s_pod_instance"].isna().any():
            plugin_data = node_data[node_data["k3s_job_name"] == plugin_name]
        else:
            plugin_data = node_data[node_data["k3s_pod_name"] == plugin_name]
        if len(plugin_data) == 0:
            errors.append(f'job {job_name} ({plugin_name}) is not running on {vsn}')
    return errors


nodes = get_manifest()
raw_jobs = get_all_jobs()
j = []
for _, v in raw_jobs.items():
    _nodes = v.get("nodes", None)
    if _nodes is None:
        _nodes = {}
    v["nodes"] = list(_nodes.keys())
    if v["state"]["last_state"] == "":
        v["state"]["last_state"] = "Unknown"
    j.append(v)
jobs = pd.json_normalize(j)

with st.sidebar:
    start_time = st.selectbox(
        "data to show",
        ("-30m", "-1h", "-3h", "-6h", "-12h", "-1d", "-7d", "-10d", "-1M", "-6M", "-1y"))
    st.button("analyze")
    if st.button("remove cache"):
        st.cache_data.clear()

    all_data = fill_completion_failure(parse_events(get_events(start_time)))
    st.header("Jobs Overview")
    st.dataframe(jobs.groupby("state.last_state")["job_id"].count().rename("jobs"))
    # st.dataframe(jobs)
    mode = st.radio(
        "",
        ("Per node", "Per plugin", "Node health", "All data for debugging"),
    )

if mode == "Per node":
    vsn = st.selectbox("VSN", [node.get("vsn") for node in nodes])
    # for index, node in enum(nodes):
        # with tabs[index]:
        # with st.expander(vsn):
    node_data = all_data[all_data["vsn"].str.lower() == vsn.lower()]
    if len(node_data) == 0:
        st.warning("no data found")
    else:
        st.subheader("Jobs reported from node")
        j = jobs[jobs["science_goal.id"].isin(node_data["goal_id"].unique().tolist())]
        st.dataframe(j)
        # node_data
        exe_time = node_data[(node_data["end_state"].str.contains("completed"))].groupby(["k3s_pod_node_name", "k3s_pod_name"])["execution_time"].describe()
        st.subheader("Execution time in seconds")
        st.dataframe(exe_time)

        st.subheader("Failed plugins")
        failed_logs = node_data[(node_data["end_state"].str.contains("failed"))][["failed_at", "k3s_pod_name", "k3s_pod_node_name", "error_log"]]
        st.dataframe(failed_logs, use_container_width=True)

        # for debugging
        st.subheader("Raw data for debugging")
        st.dataframe(node_data)
elif mode == "Per plugin":
    pass
elif mode == "Node health":
    _nodes = get_production_vsn()
    active_jobs = jobs[jobs["state.last_state"].isin(["Submitted", "Running"])]
    for bucket, df in _nodes.groupby("bucket"):
        st.subheader(bucket)
        for _, node in df.iterrows():
            def get_my_jobs(nodes, vsn):
                if vsn in nodes:
                    return True
                else:
                    return False
            my_jobs = active_jobs[active_jobs["nodes"].apply(get_my_jobs, args=(node.vsn,))]
            if len(my_jobs) == 0:
                continue
            node_data = all_data[all_data["vsn"].str.lower() == node.vsn.lower()]
            
            errors = check_node(my_jobs, node_data)
            if len(errors) > 0:
                st.subheader(node.vsn)
                st.write(errors)
                st.dataframe(node_data)
        # st.dataframe(df
    # st.dataframe(production_nodes)
elif mode == "All data for debugging":
    st.subheader("jobs")
    st.dataframe(jobs)
    st.subheader("data")
    st.dataframe(all_data)

    # job = get_job_information(job_id)
    # vsns = '|'.join(list(job["nodes"].keys()))
    # raw_events = get_events(vsns, start_time)
    # if len(raw_events) == 0:
    #     st.warning("No data found")
    # else:
    #     events = fill_completion_failure(parse_events(raw_events))
    #     if mode == "Overview":
    #         st.title("Overview")
    #         st.write(f'Job ID: {job["job_id"]}')
    #         st.write(f'Job Name: {job["name"]}')
    #         st.write(f'Number of nodes: {len(job["nodes"].keys())}')
    #         st.write(f'Status: {job["state"]["last_state"]}')
    #         st.write(f'Last updated: {job["state"]["last_updated"]}')
    #         st.header("Science Rules")
    #         for rule in job["science_rules"]:
    #             st.write(rule)

    #         runs = events[events["goal_id"] == job["science_goal"]["id"]].groupby(["vsn", "end_state"])["k3s_pod_instance"].count().unstack(fill_value=0)
    #         st.header("Executions of plugins")
    #         st.dataframe(runs)
    #     elif mode == "Nodes":
    #         vsn = st.selectbox('Nodes', tuple(job["nodes"].keys()))
    #         exe_time = events[(events["end_state"].str.contains("completed")) & (events["vsn"] == vsn) & (events["goal_id"] == job["science_goal"]["id"])].groupby(["k3s_pod_node_name", "k3s_pod_name"])["execution_time"].describe()
    #         st.header("Execution time in seconds")
    #         st.dataframe(exe_time)

    #         st.header("Raw data")
    #         st.dataframe(events)
    #     elif mode == "Plugins":
    #         plugin = st.selectbox('Plugins', tuple(p["name"] for p in job["plugins"]))
    #         if "error_log" in events:
    #             failed = events[(events["plugin_name"] == plugin) & (events["error_log"] != "")]
    #             with st.expander(f'Errors in {plugin}, if exist'):
    #                 for _, r in failed.iterrows():
    #                     st.write(f'{r["plugin_name"]} failed on {r["vsn"]} ({r["k3s_pod_node_name"]}) at {r["failed_at"]}: {r["error_log"]}')