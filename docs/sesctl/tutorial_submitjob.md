# Tutorial: Submit A Job
In this tutorial, we will submit a job to the scheduler.

## Submit an existing job
If you have followed the [previous tutorial](./tutorial_editjob.md), your job should already be created and edited in the scheduler.

To get the job ID from the scheduler,
```bash
sesctl stat
```

You should be able to see the job,
```bash
JOB_ID  NAME     USER       STATUS     START_TIME            RUNNING_TIME          
====================================================================================
17      updatedmyjob        Drafted    -                     -                     
```

To submit the job,

__NOTE: The job ID may be different for your case.__
```bash
sesctl submit -j 17
```

The scheduler would respond like,
```bash

```

## Submit a new job

The job we created has a science rule that is expected to be triggered every minute by [cronjob](https://github.com/waggle-sensor/sciencerule-checker/blob/master/docs/supported_functions.md#cronjobprogram_name-cronjob_time),
```bash
scienceRules:
- "schedule(image-sampler): cronjob('image-sampler', '* * * * *')"
```

We now think it is too frequent. And, we want to change it to every 30 minutes. Let's take the job description and change the science rule to reflect our intention. In addition to that, we also update the name of the job,
```bash
cat << EOF > updatedmyjob.yaml
---
name: updatedmyjob
plugins:
- name: image-sampler
  pluginSpec:
    image: registry.sagecontinuum.org/theone/imagesampler:0.3.0
    args:
    - -stream
    - bottom
nodes:
  W023:
scienceRules:
- "schedule(image-sampler): cronjob('image-sampler', '*/30 * * * *')"
successcriteria:
- WallClock(1d)
EOF
```

We are going to overwrite the job we already submitted with the updated job description above, we first need to know ID of the existing job,
```bash
sesctl stat
```

You should be able to see the job as follows,
```
JOB_ID  NAME     USER       STATUS     START_TIME            RUNNING_TIME          
====================================================================================
17      myjob               Created    -                     -                     
```

Then, we take the job ID (which may be different on your case) and edit the job,
```bash
sesctl edit 17 --file-path updatedmyjob.yaml
```

The scheduler would respond as,
```bash
{
 "job_id": "17",
 "status": "Drafted"
}
```

To verify if the job is edited in the scheduler,
```bash
sesctl stat
```

You should be able to see the updated job name as well as the status as "Drafted",
```
JOB_ID  NAME     USER       STATUS     START_TIME            RUNNING_TIME          
====================================================================================
17      updatedmyjob               Drafted    -                     -                     
```