# Command-line Tool For Scheduling Jobs
sesctl is design to support job submission process for users. Users can use this tool to create, edit, and submit jobs to the Sage edge scheduler (SES). Later, user can query job status and scheduling events associated to the job using the tool.

__NOTE: Users will need to explore Edge code repository (ECR) (https://portal.sagecontinuum.org) to pick edge applications they want to run before submitting a job. Users can also register their application to ECR.__

To connect to SES,
```bash
$ sesctl --server ${SES_SERVER_ADDRESS} ping
```

The scheduler in the cloud would respond as,
```bash
{
 "id": "Cloud Scheduler (cloudscheduler-sage)",
 "version": "0.18.2"
}
```

## Attributes of A Job
A job consists of attributes used to detail the job.

- `json:"job_id" yaml:"jobID"`: ID of a job given from scheduler 
- `json:"name" yaml:"name"`: user-defined name of the job
- `json:"user" yaml:"user"`: username, the owner of job
- `json:"email" yaml:"email"`: email of the user
- `json:"notification_on" yaml:"notificationOn"`: list of events for user notification
- `json:"plugins,omitempty" yaml:"plugins,omitempty"`: list of plugin specification
- `json:"node_tags" yaml:"nodeTags"`: node tags to select nodes
- `json:"nodes" yaml:"nodes"`: list of nodes
- `json:"science_rules" yaml:"scienceRules"`: user-given science rules
- `json:"success_criteria" yaml:"successCriteria"`: user-given conditions that check when the job completes

## Tutorials

1. [create job](tutorial_createjob.md) creates a job in SES

2. [edit job](tutorial_editjob.md) edits existing job in SES

2. [submit job](tutorial_submitjob.md) submits already created job in SES

3. [stat job](tutorial_statjob.md) shows status of job(s)

4. [remove job](tutorial_removejob.md) suspends and removes a job