# Tutorial: Create A Job
In this tutorial, we will create a job with a job specification already specified.

To create a job specification,
```bash
cat << EOF > myjob.yaml
---
name: myjob
plugins:
- name: imagesampler-bottom
  pluginSpec:
    image: waggle/plugin-image-sampler:0.2.5
    args:
    - -stream
    - bottom
nodes:
  W023:
successcriteria:
- WallClock(1d)
EOF
```

This specifies the intention that the user wants to run an edge application registered at `waggle/plugin-image-sampler:0.2.5` on the node named `W023` for a day.

__NOTE: Please explore Edge code repository at https://portal.sagecontinuum.org for more edge applications__

To submit the specification to SES,

```bash
$ sesctl create --file-path myjob.yaml 
{
 "job_id": "17",
 "job_name": "myjob",
 "status": "Created"
}
```

To verify if the job is submitted to SES,
```bash
$ sesctl stat
JOB_ID  NAME     USER       STATUS     START_TIME            RUNNING_TIME          
====================================================================================
17      myjob               Created    -                     -                     
```