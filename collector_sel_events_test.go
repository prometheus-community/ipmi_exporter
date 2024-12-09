package main

import (
	"github.com/prometheus-community/ipmi_exporter/freeipmi"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCollectSELEventsCollector(t *testing.T) {
	c := SELEventsCollector{}
	ch := make(chan prometheus.Metric)
	i, err := c.Collect(freeipmi.Result{}, ch, ipmiTarget{})
	assert.Nil(t, err)
	assert.Equal(t, 1, i)
}
