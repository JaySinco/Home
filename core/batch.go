package core

import (
	"strings"
)

type TaskMapping map[string]func() error

func Batch(taskSet TaskMapping) error {
	deploySet := SplitRobust(Config().Core.TaskDeploySet, ";")
	wait := make(chan taskCompleted)
	count := 0
	for _, dt := range deploySet {
		for name, task := range taskSet {
			if strings.ToLower(dt) == strings.ToLower(name) {
				count++
				Info("*** START %s ***", name)
				go func() {
					wait <- taskCompleted{
						name,
						task(),
					}
				}()
				break
			}
		}
	}
	for n := 0; n < count; n++ {
		tc := <-wait
		if tc.err != nil {
			Warn("failed to run `%s`: %v", tc.name, tc.err)
		}
		Info("*** %s END ***", tc.name)
	}
	return nil
}

type taskCompleted struct {
	name string
	err  error
}
