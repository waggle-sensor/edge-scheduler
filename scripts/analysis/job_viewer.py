import json
import urllib.request

import streamlit as st
import pandas as pd
from sage_data_client import query
import plotly.express as px

from utils import parse_events, fill_completion_failure

st.set_page_config(layout="wide")

@st.cache_data
def get_events(vsn, last):
    print("get_events called")
    return query(
    start=last,
    filter={
        "name": "sys.scheduler.status.plugin.*",
        "vsn": vsn,
        }
    )


@st.cache_data
def get_job_information(job_id):
    url = f'https://es.sagecontinuum.org/api/v1/jobs/{job_id}/status'
    contents = urllib.request.urlopen(url).read().decode()
    return json.loads(contents)


with st.sidebar:
    mode = st.radio(
        "",
        ("Overview", "Nodes", "Plugins")
    )
    job_id = st.text_input('Job ID', '')
    start_time = st.selectbox(
        "data to show",
        ("-30m", "-1h", "-3h", "-6h", "-12h", "-1d", "-7d", "-10d", "-1M", "-6M", "-1y"))
    st.button("analyze")

if job_id == "":
    st.info("Please specify job ID")
else :
    job = get_job_information(job_id)
    vsns = '|'.join(list(job["nodes"].keys()))
    raw_events = get_events(vsns, start_time)
    if len(raw_events) == 0:
        st.warning("No data found")
    else:
        events = fill_completion_failure(parse_events(raw_events))
        if mode == "Overview":
            st.title("Overview")
            st.write(f'Job ID: {job["job_id"]}')
            st.write(f'Job Name: {job["name"]}')
            st.write(f'Number of nodes: {len(job["nodes"].keys())}')
            st.write(f'Status: {job["state"]["last_state"]}')
            st.write(f'Last updated: {job["state"]["last_updated"]}')
            st.header("Science Rules")
            for rule in job["science_rules"]:
                st.write(rule)

            runs = events[events["goal_id"] == job["science_goal"]["id"]].groupby(["vsn", "end_state"])["k3s_pod_instance"].count().unstack(fill_value=0)
            st.header("Executions of plugins")
            st.dataframe(runs)
        elif mode == "Nodes":
            vsn = st.selectbox('Nodes', tuple(job["nodes"].keys()))
            exe_time = events[(events["end_state"].str.contains("completed")) & (events["vsn"] == vsn) & (events["goal_id"] == job["science_goal"]["id"])].groupby(["k3s_pod_node_name", "k3s_pod_name"])["execution_time"].describe()
            st.header("Execution time in seconds")
            st.dataframe(exe_time)

            st.header("Raw data")
            st.dataframe(events)
        elif mode == "Plugins":
            plugin = st.selectbox('Plugins', tuple(p["name"] for p in job["plugins"]))
            if "error_log" in events:
                failed = events[(events["plugin_name"] == plugin) & (events["error_log"] != "")]
                with st.expander(f'Errors in {plugin}, if exist'):
                    for _, r in failed.iterrows():
                        st.write(f'{r["plugin_name"]} failed on {r["vsn"]} ({r["k3s_pod_node_name"]}) at {r["failed_at"]}: {r["error_log"]}')