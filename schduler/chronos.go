package scheduler

import (
	"github.com/behance/go-chronos/chronos"
)

//ChronosScheduler is chronos scheduler client
type chronosScheduler struct {
	client chronos.Chronos
}

func NewChronosScheduler(url string) Scheduler {
	config := chronos.Config{
		URL: url,
	}

	client, _ := chronos.NewClient(config)
	return &chronosScheduler{client: client}
}

//TODO implements chronos scheduler

func (c *chronosScheduler) AddOnceJob(...Job) error {
	return nil
}

func (c *chronosScheduler) AddScheduledJob(job Job) error {
	//jobs := chronos.Job{}
	jobs := transportJob(job)

	err := c.client.AddScheduledJob(jobs)
	if err != nil {
		return err
	}
	return nil
}

func (c *chronosScheduler) DeleteJob(name string) error {
	err := c.client.DeleteJob(name)
	if err != nil {
		return err
	}
	return nil
}

func (c *chronosScheduler) CancelJob(name string) error {
	return nil
}

func transportJob(job Job) *chronos.Job {
	req := job
	schedule, _ := chronos.FormatSchedule(job.StartTime, "PT2M", "R1")

	jobs := &chronos.Job{
		Name:     req.Name,
		Schedule: schedule,
		Command:  req.Command,
	}
	return jobs
}
