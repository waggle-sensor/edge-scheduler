# Sage Edge Scheduler (SES)
#### Lead: Kate Keahey and Yongho Kim

### Overview:

All configuration changes, whether they be software updates or new edge computing algorithms are handled by the SES.  Users who have edge code already running and deployed on Waggle nodes can use their authentication token to push configuration changes to nodes via the SES.  Users can also submit “jobs” that can be scheduled and run on nodes at a later time.  The SES makes all configuration and system update decisions, and queues up changes that can be pushed out to nodes when they contact Beehive.  

### Requirements:

Like the SLT, the SES uses token-based authentication.  An automated function, using the token, can submit jobs to the edge computing queue or make configuration changes to the running job.  The SES must also maintain a queue of jobs, manage priorities, and make decisions on evicting edge computation if needed.  The SES also compares resource requirements as provided by the ECR to available resources on nodes.

### Milestones:
* Publish design document, including examples
* Deploy auth-tokens and single tenant queue to Sage network
* Deploy multi-tenant SES and WES
* Release V1.0 of SES

