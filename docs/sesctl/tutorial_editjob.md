# Tutorial: edit a job
In this tutorial, we will modify the exiting job we created in the [previous tutorial](tutorial_createjob.md).

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
JOB_ID  NAME                USER       STATUS     AGE     
====================================================================
...
17      myjob               yonghokim  Created    
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
```bash
JOB_ID  NAME                USER       STATUS     AGE     
====================================================================
...
17      updatedmyjob        yonghokim  Drafted   
```