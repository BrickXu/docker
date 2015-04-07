package daemon

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/docker/docker/engine"
)

func (daemon *Daemon) ContainerTop(job *engine.Job) error {
	if len(job.Args) != 1 && len(job.Args) != 2 {
		return fmt.Errorf("Not enough arguments. Usage: %s CONTAINER [PS_ARGS]\n", job.Name)
	}
	var (
		name   = job.Args[0]
		psArgs = "-ef"
	)

	if len(job.Args) == 2 && job.Args[1] != "" {
		psArgs = job.Args[1]
	}

	container, err := daemon.Get(name)
	if err != nil {
		return err
	}
	if !container.IsRunning() {
		return fmt.Errorf("Container %s is not running", name)
	}
	pids, err := daemon.ExecutionDriver().GetPidsForContainer(container.ID)
	if err != nil {
		return err
	}
	output, err := exec.Command("ps", strings.Split(psArgs, " ")...).Output()
	if err != nil {
		return fmt.Errorf("Error running ps: %s", err)
	}

	lines := strings.Split(string(output), "\n")
	header := strings.Fields(lines[0])
	out := &engine.Env{}
	out.SetList("Titles", header)

	pidIndex := -1
	for i, name := range header {
		if name == "PID" {
			pidIndex = i
		}
	}
	if pidIndex == -1 {
		return fmt.Errorf("Couldn't find PID field in ps output")
	}

	processes := [][]string{}
	for _, line := range lines[1:] {
		if len(line) == 0 {
			continue
		}
		fields := strings.Fields(line)
		p, err := strconv.Atoi(fields[pidIndex])
		if err != nil {
			return fmt.Errorf("Unexpected pid '%s': %s", fields[pidIndex], err)
		}

		for _, pid := range pids {
			if pid == p {
				// Make sure number of fields equals number of header titles
				// merging "overhanging" fields
				process := fields[:len(header)-1]
				process = append(process, strings.Join(fields[len(header)-1:], " "))
				processes = append(processes, process)
			}
		}
	}
	out.SetJson("Processes", processes)
	out.WriteTo(job.Stdout)
	return nil
}
