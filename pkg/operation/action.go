package operator

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/pingcap-incubator/tiops/pkg/executor"
	"github.com/pingcap-incubator/tiops/pkg/meta"
	"github.com/pingcap-incubator/tiops/pkg/module"
	"github.com/pingcap/errors"
)

var defaultTimeoutForReady = time.Second * 60

func filterComponent(comps []meta.Component, component string) (res []meta.Component) {
	if component == "" {
		res = comps
		return
	}

	for _, c := range comps {
		if c.Name() != component {
			continue
		}

		res = append(res, c)
	}

	return
}

func filterInstance(instances []meta.Instance, node string) (res []meta.Instance) {
	if node == "" {
		res = instances
		return
	}

	for _, c := range instances {
		if c.GetIP() != node {
			continue
		}

		res = append(res, c)
	}

	return
}

// ExecutorGetter get the executor by host.
type ExecutorGetter interface {
	Get(host string) (e executor.TiOpsExecutor)
}

// Start the cluster.
func Start(
	getter ExecutorGetter,
	w io.Writer,
	spec *meta.Specification,
	component string,
	node string,
) error {
	coms := spec.ComponentsByStartOrder()
	coms = filterComponent(coms, component)

	for _, com := range coms {
		err := StartComponent(getter, w, filterInstance(com.Instances(), node))
		if err != nil {
			return errors.Annotatef(err, "failed to start %s", com.Name())
		}
	}

	return nil
}

// Stop the cluster.
func Stop(
	getter ExecutorGetter,
	w io.Writer,
	spec *meta.Specification,
	component string,
	node string,
) error {
	coms := spec.ComponentsByStartOrder()
	coms = filterComponent(coms, component)

	for _, com := range coms {
		err := StopComponent(getter, w, filterInstance(com.Instances(), node))
		if err != nil {
			return errors.Annotatef(err, "failed to stop %s", com.Name())
		}
	}
	return nil
}

// Restart the cluster.
func Restart(
	getter ExecutorGetter,
	w io.Writer,
	spec *meta.Specification,
	component string,
	node string,
) error {
	coms := spec.ComponentsByStartOrder()
	coms = filterComponent(coms, component)

	for _, com := range coms {
		err := StopComponent(getter, w, filterInstance(com.Instances(), node))
		if err != nil {
			return errors.Annotatef(err, "failed to stop %s", com.Name())
		}

		err = StartComponent(getter, w, filterInstance(com.Instances(), node))
		if err != nil {
			return errors.Annotatef(err, "failed to start %s", com.Name())
		}
	}

	return nil
}

// Destroy the cluster.
func Destroy(
	getter ExecutorGetter,
	w io.Writer,
	spec *meta.Specification,
	component string,
	node string,
) error {

	return nil
}

// StartComponent start the instances.
func StartComponent(getter ExecutorGetter, w io.Writer, instances []meta.Instance) error {
	if len(instances) <= 0 {
		return nil
	}

	name := instances[0].ComponentName()
	fmt.Fprintf(w, "Starting component %s", name)

	for _, ins := range instances {
		e := getter.Get(ins.GetIP())
		fmt.Fprintf(w, "Starting instance %s", ins.GetIP())

		// Start by systemd.
		c := module.SystemdModuleConfig{
			Unit:   ins.ServiceName(),
			Action: "start",
			// Scope: "",
		}
		systemd := module.NewSystemdModule(c)
		stdout, stderr, err := systemd.Execute(e)

		io.Copy(w, bytes.NewReader(stdout))
		io.Copy(w, bytes.NewReader(stderr))

		if err != nil {
			return errors.Annotatef(err, "failed to start: %s", ins.GetIP())
		}

		// Check ready.
		err = ins.Ready(e)
		if err != nil {
			str := fmt.Sprintf("%s failed to start: %s", ins.GetIP(), err)
			fmt.Fprintln(w, str)
			return errors.Annotatef(err, str)
		}

		fmt.Fprintf(w, "Start %s success", ins.GetIP())
	}

	return nil
}

// StopComponent stop the instances.
func StopComponent(getter ExecutorGetter, w io.Writer, instances []meta.Instance) error {
	if len(instances) <= 0 {
		return nil
	}

	name := instances[0].ComponentName()
	fmt.Fprintf(w, "Stopping component %s", name)

	for _, ins := range instances {
		e := getter.Get(ins.GetIP())
		fmt.Fprintf(w, "Stopping instance %s", ins.GetIP())

		// Stop by systemd.
		c := module.SystemdModuleConfig{
			Unit:   ins.ServiceName(),
			Action: "stop",
			// Scope: "",
		}
		systemd := module.NewSystemdModule(c)
		stdout, stderr, err := systemd.Execute(e)

		io.Copy(w, bytes.NewReader(stdout))
		io.Copy(w, bytes.NewReader(stderr))

		if err != nil {
			return errors.Annotatef(err, "failed to stop: %s", ins.GetIP())
		}

		err = ins.Ready(e)
		if err != nil {
			str := fmt.Sprintf("%s failed to stop: %s", ins.GetIP(), err)
			fmt.Fprintln(w, str)
			return errors.Annotatef(err, str)
		}

		fmt.Fprintf(w, "Stop %s success", ins.GetIP())
	}

	return nil
}

// PrintClusterStatus print cluster status into the io.Writer.
func PrintClusterStatus(getter ExecutorGetter, w io.Writer, spec *meta.Specification) (health bool) {
	// TODO

	return true
}
