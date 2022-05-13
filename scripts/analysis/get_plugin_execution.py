import sys
import os

import sage_data_client

vsn = sys.argv[1]
print(vsn)

df = sage_data_client.query(
    start="-1h",
    filter={
        "name": "sys.scheduler.status.plugin.*",
        "vsn": vsn,
    }
)

# Filter only events related to plugin execution
launched_plugins = df[df.name.str.contains("launched")]
complete_plugins = df[df.name.str.contains("complete")]
failed_plugins = df[df.name.str.contains("failed")]
# for row in df.iterrow():
#     row.