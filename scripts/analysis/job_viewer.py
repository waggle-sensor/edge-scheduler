import json
import urllib.request

import streamlit as st
import pandas as pd
from sage_data_client import query

st.set_page_config(layout="wide")


def parse_events(df):
    v = []
    for _, row in df.iterrows():
        r = json.loads(row.value)
        r["timestamp"] = row.timestamp.isoformat()
        r["node"] = row["meta.node"]
        r["vsn"] = row["meta.vsn"]
        r["event"] = row["name"]
        v.append(r)
    return pd.read_json(json.dumps(v))


def fill_completion_failure(df):
    # Filter only events related to plugin execution
    launched = df[df.event.str.contains("launched")]
    completed = df[df.event.str.contains("complete")]
    failed = df[df.event.str.contains("failed")]
    # launched.loc[launched["k3s_pod_name"] == completed["k3s_pod_name"]]
    for index, p in launched.iterrows():
        found = completed[completed.k3s_pod_instance == p.k3s_pod_instance]
        if len(found) > 0:
            launched.loc[index, "completed_at"] = found.iloc[0].timestamp
            launched.loc[index, "execution_time"] = (found.iloc[0].timestamp - p.timestamp).total_seconds()
            launched.loc[index, "k3s_pod_node_name"] = found.iloc[0].k3s_pod_node_name
            launched.loc[index, "end_state"] = "completed"
        else:
            found = failed[failed.k3s_pod_name == p.k3s_pod_name]
            if len(found) > 0:
                launched.loc[index, "failed_at"] = found.iloc[0].timestamp
                launched.loc[index, "reason"] = found.iloc[0].reason
                if "error_log" in found.iloc[0]:
                    launched.loc[index, "error_log"] = found.iloc[0]["error_log"]
                launched.loc[index, "end_state"] = "failed"
            else:
                launched.loc[index, "end_state"] = "unknown"
    return launched


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


def update_view(job, data_to_show):
    

    tab_overview, tab_nodes, tab_plugins = st.tabs(["Overview", "Nodes", "Plugins"])
    with tab_overview:
        st.title("Overview")
        st.write(f'Job ID: {job["job_id"]}')
        st.write(f'Job Name: {job["name"]}')
        st.write(f'Number of nodes: {len(job["nodes"].keys())}')
        st.write(f'Status: {job["state"]["last_state"]}')
        st.write(f'Last updated: {job["state"]["last_updated"]}')

        runs = events.groupby(["vsn", "end_state"])["k3s_pod_instance"].count().unstack(fill_value=0)
        st.header("Executions of plugins")
        st.dataframe(runs)

    with tab_nodes:
        vsn = st.selectbox('Nodes', tuple(job["nodes"].keys()))
        print(vsn)
        exe_time = events[(events["end_state"].str.contains("completed")) & (events["vsn"] == vsn)].groupby(["k3s_pod_node_name", "k3s_pod_name"])["execution_time"].describe()
        st.header("Execution time in seconds")
        st.dataframe(exe_time)

        st.header("Raw data")
        st.dataframe(events)



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

            runs = events.groupby(["vsn", "end_state"])["k3s_pod_instance"].count().unstack(fill_value=0)
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