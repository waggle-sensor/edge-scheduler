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
def get_perf(plugin, last):
    return query(
        start=last,
        bucket="grafana-agent",
        filter={
            "container": plugin,
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
    debug = st.checkbox("debug")

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
            plugin_name = st.selectbox('Plugins', tuple(p["name"] for p in job["plugins"]))
            plugin_events = events[events["plugin_name"] == plugin_name]
            plugin_perf = get_perf(plugin_name, start_time)
            if debug:
                st.dataframe(plugin_events)
                st.dataframe(plugin_perf)
            for _, e in plugin_events.iterrows():
                st.write(f'{e["k3s_pod_instance"]} ran on {e["vsn"]} ({e["k3s_pod_node_name"]})')
                if e["end_state"] == "failed":
                    st.error(f'{e["reason"]}: {e["error_log"]}')
                d = int(e["execution_time"]) if e["execution_time"] > 0 else 0
                mask = (e["vsn"] == plugin_perf["meta.vsn"]) & \
                       (plugin_perf["timestamp"] >= e["timestamp"]) & \
                       (plugin_perf["timestamp"] < e["timestamp"] + pd.Timedelta(d, unit='s'))
                perf = plugin_perf[mask]
                if len(perf) == 0:
                    st.warning("no performance data found")
                else:
                    if debug:
                        st.dataframe(perf)
                    last_cpu_record = perf[perf["name"] == "container_cpu_usage_seconds_total"].sort_values("value").iloc[-1]
                    # we assume 100 ms CPU time per second
                    util = last_cpu_record["value"] / (last_cpu_record["timestamp"] - e["timestamp"]).total_seconds() / 0.1 * 100.

                    mem = pd.to_numeric(perf[perf["name"] == "container_memory_working_set_bytes"]["value"]).mean()
                    col1, col2, col3 = st.columns(3)
                    col1.metric("Averaged CPU (%)", util, "")
                    col2.metric("Averaged Memory (MB)", mem/1e6, "")
                    col3.metric("Execution time (seconds)", e["execution_time"], "")
                st.divider()
