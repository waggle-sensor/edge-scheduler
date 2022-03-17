# Command-line Tool For Scheduling Jobs
sesctl is design to support job submission process for users. Users can use this tool to create, edit, and submit jobs to the Sage edge scheduler (SES). Later, user can query job status and scheduling events associated to the job using the tool.

__NOTE: Users will need to explore Edge code repository (ECR) (https://portal.sagecontinuum.org) to pick edge applications they want to run before submitting a job. Users can also register their application to ECR.__

To connect to SES,
```bash
$ sesctl --server ${SES_SERVER_ADDRESS} ping
{
 "id": "Cloud Scheduler (ses-sage)",
 "version": "0.9.8"
}
```

# Tutorials

1. [create_job](tutorial_createjob.md) creates a job in SES

2. [submit_job](tutorial_submitjob.md) submits already created job in SES

3. [status_job](tutorial_statjob.md) shows status of job(s)

4. [delete_job](tutorial_deletejob.md) suspends and deletes a job