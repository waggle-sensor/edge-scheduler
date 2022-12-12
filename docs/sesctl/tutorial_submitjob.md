# Tutorial: submit a job
In this tutorial, we will submit a job to the scheduler.

## Submit an existing job
If you have followed the [previous tutorial](./tutorial_editjob.md), your job should already be created and edited in the scheduler.

To get the job ID from the scheduler,
```bash
sesctl stat
```

You should be able to see the job,
```bash
JOB_ID  NAME                USER       STATUS     AGE     
====================================================================
...
17      updatedmyjob        yonghokim  Drafted    
```

To submit the job,

__NOTE: The job ID may be different for your case.__
```bash
sesctl submit --job-id 17
```

The scheduler would respond like,
```bash
{
 "job_id": "17",
 "state": "Submitted"
}
```

You should be able to see that the job is in either "Submitted" or "Running" state, depending on whether the node already picked up your job to schedule and run,
```bash
JOB_ID  NAME                USER       STATUS     AGE     
====================================================================
...
17      updatedmyjob        yonghokim  Running    13s            
```

The node would schedule the image sampler plugin every 30 minutes. As a result of the schedule, you will see image samples from [query-browser](https://portal.sagecontinuum.org/query-browser). You can also follow the [how to access large files](https://docs.waggle-edge.ai/docs/tutorials/accessing-data#accessing-large-files-ie-training-data) tutorial to get the samples.

## Submit a new job
The `submit` subcommand also accepts a job description and create a new job for it. This is more convenient than people do the 2 steps for create and submit a job. The following demonstrates how one can create a job file and submit it without creating or editing the job in the scheduler.

To create a new job,
```bash
cat << EOF > mynewjob.yaml
---
name: mynewjob
plugins:
- name: new-image-sampler
  pluginSpec:
    image: registry.sagecontinuum.org/theone/imagesampler:0.3.0
    args:
    - -stream
    - bottom
nodes:
  W023:
scienceRules:
- "schedule(new-image-sampler): cronjob('new-image-sampler', '0 * * * *')"
successcriteria:
- WallClock(1d)
EOF
```

To submit a new job from above job description,
```bash
sesctl submit --file-path mynewjob.yaml
```

You should be able to see that the job is created and assigned a new job ID,
```bash
JOB_ID  NAME                USER       STATUS     AGE     
====================================================================
...
18      mynewjob            yonghokim  Running    20s            
```