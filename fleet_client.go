package main

import (
	"fmt"
	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/job"
	"log"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type FleetClient struct {
	Client *client.API
}

func NewFleetAPIClient(path string) (*FleetClient, error) {
	ep, _ := url.Parse(path)
	dialUnix := ep.Scheme == "unix" || ep.Scheme == "file"
	dialFunc := net.Dial

	if dialUnix {
		ep.Host = "domain-sock"
		ep.Scheme = "http"

		sockPath := ep.Path

		ep.Path = ""

		dialFunc = func(string, string) (net.Conn, error) {
			return net.Dial("unix", sockPath)
		}
	}

	tr := &http.Transport{
		Dial: dialFunc,
	}

	cl := http.Client{Transport: tr}

	httpClient, err := client.NewHTTPClient(&cl, *ep)

	if err != nil {
		return nil, err
	} else {
		return &FleetClient{&httpClient}, nil
	}
}

func (c *FleetClient) FindMatchingUnits(name string) []string {
	if exist, _ := c.AssertUnitExistence(name); exist {
		return []string{name}
	} else {
		units, _ := c.SubUnits(name)

		return units
	}
}

func (c *FleetClient) AssertUnitExistence(name string) (bool, error) {
	unit, err := (*c.Client).Unit(name)

	if unit != nil {
		return true, err
	} else {
		return false, err
	}
}

func (c *FleetClient) SubUnits(name string) ([]string, error) {
	units, err := (*c.Client).Units()

	if err != nil {
		return []string{}, err
	}

	subUnitRegexp, err := regexp.Compile(
		fmt.Sprintf("^%s@.+", strings.TrimSuffix(name, ".service")),
	)

	if err != nil {
		return []string{}, err
	}

	result := []string{}

	for _, unit := range units {
		if subUnitRegexp.MatchString(unit.Name) {
			result = append(result, unit.Name)
		}
	}

	return result, nil
}

func (c *FleetClient) WaitUntilTargetStateReached(name, state string) {
	sleep := 500 * time.Millisecond

	for {
		unit, err := (*c.Client).Unit(name)

		if err != nil {
			log.Printf("Error  during the wait %s", err.Error())
		}

		if unit != nil {
			log.Printf(
				"Current state of %s is %s but the wanted is %s",
				unit.Name,
				unit.CurrentState,
				state,
			)
		}

		if err == nil && unit.CurrentState == state {
			return
		}

		time.Sleep(sleep)
	}
}

func (c *FleetClient) SwapStateUnit(name string, beforeState, afterState job.JobState) error {
	if err := (*c.Client).SetUnitTargetState(name, string(beforeState)); err != nil {
		return err
	}

	c.WaitUntilTargetStateReached(name, string(beforeState))

	if err := (*c.Client).SetUnitTargetState(name, string(afterState)); err != nil {
		log.Printf("Moving %s to the %s state : failed because %s", name, string(afterState), err.Error())

		return err
	}

	c.WaitUntilTargetStateReached(name, string(afterState))

	return nil
}

func (c *FleetClient) RestartUnit(name string) error {
	return c.SwapStateUnit(name, job.JobStateLoaded, job.JobStateLaunched)
}

func (c *FleetClient) RebalanceUnit(name string) error {
	return c.SwapStateUnit(name, job.JobStateInactive, job.JobStateLaunched)
}
