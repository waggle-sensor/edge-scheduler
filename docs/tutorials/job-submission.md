# Job Submission

This tutorial lets you script a Sage job and submit it to the system to run the job.

## Node Selection
First, you need to list nodes on which edge applications will be run. The [Sage portal](https://portal.sagecontinuum.org/nodes) is a good place to explore the nodes and their location. Each node may have different types of sensors; you can check the list of sensors on a node in from the portal, [W023](https://portal.sagecontinuum.org/node/W023?tab=sensors) for example.

## Application Selection
Based on the nodes and their available sensors, you choose one or more applications from the [edge code repository](https://portal.sagecontinuum.org/apps/explore). Note that the application may require different sensors from the nodes. You should be able to check those requirements from their science overview page, e.g. [cloud cover](https://portal.sagecontinuum.org/apps/app/seonghapark/cloud-cover).

## Create a job
In this tutorial we choose a few applications, each of them doing,
- cloud-cover: calculates the percantage of cloud from the sky and outputs the ratio of the cloud ranging from [0., 1.], 1. for full cloud and 0. for clear sky.
- object-counter: reports counts of any recognized objects from a camera image.
- motion-detection: simply reports whether there is motion in the camera view; 1 for motion detected, 0 for not.

```yaml
name: tutorial-job
plugins:
- name: cloud-cover-myjob
  pluginSpec:
    image: registry.sagecontinuum.org/seonghapark/cloud-cover:0.1.3
    args:
    - -stream
    - top_camera
    selector:
      resource.gpu: "true"
- name: object-counter-myjob
  pluginSpec:
    image: registry.sagecontinuum.org/yonghokim/object-counter:0.5.1
    args:
    - -stream
    - bottom_camera
    - -all-objects
    selector:
      resource.gpu: "true"
- name: motion-detection-myjob
  pluginSpec:
    image: registry.sagecontinuum.org/seonghapark/motion-detector:0.3.0
    args:
    - --input
    - bottom_camera
nodes:
  W0AF: true
  W0B2: true
  W0B3: true
  W0B4: true
  W0B5: true
  W0B6: true
  W0B7: true
  W0B8: true
  W0B9: true
  W0BA: true
scienceRules:
- 'schedule(cloud-cover-myjob): cronjob("cloud-cover-myjob", "*/5 * * * *")'
- 'schedule(object-counter-myjob): cronjob("object-counter-myjob", "*/2 * * * * *")'
- 'schedule(motion-detection-myjob): cronjob("motion-detection-myjob", "* * * * *")'
```

Once you are signed in to the Sage portal, go to the [job submission page](https://portal.sagecontinuum.org/create-job) and paste the job description above.

> WARNING: You will need the permission to submit the job to the nodes listed in the job. Please [contact us](https://sagecontinuum.org/docs/contact-us) for the permission.

Once the submission goes through the system, you can see the job in the [portal](https://portal.sagecontinuum.org/jobs/my-jobs) as well.