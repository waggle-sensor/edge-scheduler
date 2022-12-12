# Tutorial: get details of a job
Once your job is submitted you may want to check how the job is being served by Waggle nodes.

__NOTE: This has many rooms to improve in order to provide in-depth information about jobs. This will be updated frequently as we bring more functionalities to the tool__

To see status of a job,
```bash
sesctl stat --job-id 18
```

The `stat` subcommand takes status of the job from the scheduler and shows information that are useful for users to know,
```bash
===== JOB STATUS =====
Job ID: 18
Job Name: updatedmyjob
Job Owner: yonghokim
Job Status: Running
  Last updated: 2022-12-12 15:49:10.177755884 +0000 UTC
  Submitted: 2022-12-12 15:49:10.124140334 +0000 UTC
  Started: 2022-12-12 15:49:10.177755884 +0000 UTC

===== SCHEDULING DETAILS =====
Science Goal ID: 541a0db4-2332-4d23-558c-e1a82569e64c
Total number of nodes 1
```

One importans bit of information from the status is the science goal ID. Since all the plugins will publish data with the goal ID, the ID will be necessity to query data produced under the job. More information about how to query data from Waggle are descirbed in the [data API tutorial](https://docs.waggle-edge.ai/docs/tutorials/accessing-data#using-the-data-api).