# Tutorial: suspend or remove an existing job
You may suspend or remove your job from the scheduler when you think it should.

To suspend the job we submitted from the [submit tutorial](./tutorial_submitjob.md),
```bash
sesctl rm --suspend 18
```

The scheduler would response like,
```bash
{
 "job_id": "18",
 "state": "Suspended"
}
```

To verify that the job is suspended in the scheduler,
```bash
sesctl stat
```

And, uou should be able to see that the job is now suspended,
```bash
JOB_ID  NAME                USER       STATUS     AGE     
====================================================================
...
18      mynewjob            yonghokim  Suspended        
```

Suspending a job means that the nodes that were serving the job drop the job from the list. Though, the job is still shown in the cloud scheduler.

If you want to remove the job from the scheduler,
```bash
sesctl rm 18
```

The scheduler would response like,
```bash
{
 "job_id": "18",
 "state": "Removed"
}
```

Since the job is in removed state, the job is no longer shown in the `sesctl stat` subcommand. However, the scheduler still holds your job in the scheduler. So, you can check it by executing the `stat` subcommand with the `--show-all` argument,
```bash
sesctl stat --show-all
```

You will see the removed job,
```bash
JOB_ID  NAME                USER       STATUS     AGE     
====================================================================
...
18      mynewjob            yonghokim  Removed         
```