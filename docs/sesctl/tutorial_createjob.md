# Tutorial: Create A Job
In this tutorial, we will create a job with a job description.

To create a job description,
```bash
cat << EOF > myjob.yaml
---
name: myjob
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
- "schedule(image-sampler): cronjob('image-sampler', '* * * * *')"
successcriteria:
- WallClock(1d)
EOF
```

The job specifies the intention that the user wants to run an edge application registered at `registry.sagecontinuum.org/theone/imagesampler:0.3.0` on the node named `W023` for a day.

__NOTE: Please explore Edge code repository at https://portal.sagecontinuum.org for more edge applications__

To upload the job description to the scheduler,
```bash
sesctl create --file-path myjob.yaml
```

The scheduler would respond as,
```bash
{
 "job_id": "17",
 "job_name": "myjob",
 "status": "Created"
}
```

To verify if the job is created in the scheduler,
```bash
sesctl stat
```

You should be able to see an entry of the job as follows,
```
JOB_ID  NAME     USER       STATUS     START_TIME            RUNNING_TIME          
====================================================================================
17      myjob               Created    -                     -                     
```