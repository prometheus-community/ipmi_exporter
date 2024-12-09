package main

import (
	"bytes"
	"fmt"
	"github.com/prometheus-community/ipmi_exporter/freeipmi"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"log"
	"log/slog"
	"testing"
)

type withExemplarsMetric struct {
	prometheus.Metric

	exemplars []*dto.Exemplar
}

func TestCollectSELEventsCollector(t *testing.T) {
	testCases := []struct {
		arg       string
		wantState string
		wantGauge float64
	}{
		{
			arg:       "1,Mar-01-2024,17:00:11,SEL,Event Logging Disabled,Nominal,Log Area Reset/Cleared\n",
			wantState: "Nominal",
			wantGauge: 1,
		},
		{
			arg:       "2,Aug-05-2024,14:31:52,System Board Intrusion,Physical Security,Critical,General Chassis Intrusion ; Intrusion while system Off\n",
			wantState: "Critical",
			wantGauge: 1,
		},
	}

	var logBuf bytes.Buffer
	log.SetOutput(&logBuf)
	slogger := slog.New(slog.NewTextHandler(&logBuf, nil))

	for n, tc := range testCases {
		t.Run(fmt.Sprintf("test%d", n), func(t *testing.T) {
			ch := make(chan prometheus.Metric)
			defer close(ch)

			c := SELEventsCollector{}

			go func() {
				i, err := c.Collect(
					freeipmi.Execute("/bin/echo", []string{tc.arg}, "", "", slogger),
					ch,
					ipmiTarget{},
				)
				assert.Nil(t, err)
				assert.Equal(t, 1, i)
			}()

			metric := <-ch
			dm := dto.Metric{}
			err := metric.Write(&dm)

			assert.Nil(t, err)
			assert.Equal(t, *dm.Gauge.Value, tc.wantGauge)
			assert.Equal(t, len(dm.Label), 1)
			assert.Equal(t, *dm.Label[0].Name, "state")
			assert.Equal(t, *dm.Label[0].Value, tc.wantState)
		})
	}
}
