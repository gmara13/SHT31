package sht3x

import (
	"encoding/binary"
	"errors"
	"time"

	i2c "github.com/d2r2/go-i2c"
	"github.com/davecgh/go-spew/spew"
)

// Command byte's sequences
var (
	// Measure values in "single shot mode".
	CMD_SINGLE_MEASURE_HIGH_CSE   = []byte{0x2C, 0x06} // Single Measure of Temp. and Hum.; High precise; Clock stretching enabled
	CMD_SINGLE_MEASURE_MEDIUM_CSE = []byte{0x2C, 0x0D} // Single Measure of Temp. and Hum.; Medium precise; Clocl stretching enabled
	CMD_SINGLE_MEASURE_LOW_CSE    = []byte{0x2C, 0x10} // Single Measure of Temp. and Hum.; Low precise; Clock stretching enabled
	CMD_SINGLE_MEASURE_HIGH       = []byte{0x24, 0x00} // Single Measure of Temp. and Hum.; High precise
	CMD_SINGLE_MEASURE_MEDIUM     = []byte{0x24, 0x0B} // Single Measure of Temp. and Hum.; Medium precise
	CMD_SINGLE_MEASURE_LOW        = []byte{0x24, 0x16} // Single Measure of Temp. and Hum.; Low precise
// Other commands.
	CMD_PERIOD_FETCH = []byte{0xE0, 0x00} // Read data after being measured by periodic acquisition mode command
	CMD_ART          = []byte{0x2B, 0x32} // Activate "accelerated response time"
	CMD_BREAK        = []byte{0x30, 0x93} // Interrupt "periodic acqusition mode" and return to "single shot mode"
	CMD_RESET        = []byte{0x30, 0xA2} // Soft reset command

)

// MeasureRepeatability used to define measure precision.
type MeasureRepeatability int
type PeriodicMeasure int

const (
	RepeatabilityLow    MeasureRepeatability = iota + 1 // Low precision
	RepeatabilityMedium                                 // Medium precision
	RepeatabilityHigh                                   // High precision
)

// String define stringer interface.
func (v MeasureRepeatability) String() string {
	switch v {
	case RepeatabilityLow:
		return "Measure Repeatability Low"
	case RepeatabilityMedium:
		return "Measure Repeatability Medium"
	case RepeatabilityHigh:
		return "Measure Repeatability High"
	default:
		return "<unknown>"
	}
}


// GetMeasureTime define how long to wait for the measure process
// to complete according to specification.
func (v MeasureRepeatability) GetMeasureTime() time.Duration {
	switch v {
	case RepeatabilityLow:
		return 4500 * time.Microsecond
	case RepeatabilityMedium:
		return 6500 * time.Microsecond
	case RepeatabilityHigh:
		return 15500 * time.Microsecond
	default:
		return 0
	}
}

// SHT3X is a sensor itself.
type SHT3X struct {
	lastStatusReg *uint16
	lastCmd       []byte
	lastPeriodic  PeriodicMeasure
	lastPrecision MeasureRepeatability
}

// NewSHT3X return new sensor instance.
func NewSHT3X() *SHT3X {
	v := &SHT3X{}
	return v
}


// readDataWithCRCCheck read block of data which ordinary contain
// uncompensated temperature and humidity values.
func (v *SHT3X) readDataWithCRCCheck(i2c *i2c.I2C, blockCount int) ([]uint16, error) {
	const blockSize = 2 + 1
	data := make([]struct {
		Data [2]byte
		CRC  byte
	}, blockCount)

	err := readDataToStruct(i2c, blockSize*blockCount, binary.BigEndian, data)
	if err != nil {
		return nil, err
	}
	var results []uint16
	for i := 0; i < blockCount; i++ {
		calcCRC := calcCRC_SHT3X(0xFF, data[i].Data[:2])
		crc := data[i].CRC
		if calcCRC != crc {
			err := errors.New(spew.Sprintf(
				"CRCs doesn't match: CRC from sensor (0x%0X) != calculated CRC (0x%0X)",
				crc, calcCRC))
			return nil, err
		} else {
			lg.Debugf("CRCs verified: CRC from sensor (0x%0X) = calculated CRC (0x%0X)",
				crc, calcCRC)
		}
		results = append(results, getU16BE(data[i].Data[:2]))

	}
	return results, nil
}

// initiateMeasure used to initiate temperature and humidity measurement process.
func (v *SHT3X) initiateMeasure(i2c *i2c.I2C, cmd []byte,
	precision MeasureRepeatability) error {

	_, err := i2c.WriteBytes(cmd)
	if err != nil {
		return err
	}
	v.lastCmd = cmd

	// Wait according to conversion time specification
	pause := precision.GetMeasureTime()
	time.Sleep(pause)
	return nil
}

// ReadUncompTemperatureAndHumidity returns uncompensated humidity and
// temperature obtained from sensor in "single shot mode".
func (v *SHT3X) ReadUncompTemperatureAndHumidity(i2c *i2c.I2C,
	precision MeasureRepeatability) (uint16, uint16, error) {

	lg.Debug("Measuring temperature and humidity...")
	var cmd []byte
	switch precision {
	case RepeatabilityLow:
		cmd = CMD_SINGLE_MEASURE_LOW
	case RepeatabilityMedium:
		cmd = CMD_SINGLE_MEASURE_MEDIUM
	case RepeatabilityHigh:
		cmd = CMD_SINGLE_MEASURE_HIGH
	}
	err := v.initiateMeasure(i2c, cmd, precision)
	if err != nil {
		return 0, 0, err
	}

	data, err := v.readDataWithCRCCheck(i2c, 2)
	if err != nil {
		return 0, 0, err
	}
	return data[0], data[1], nil
}

// ReadTemperatureAndRelativeHumidity returns humidity and
// temperature obtained from sensor in "single shot mode".
func (v *SHT3X) ReadTemperatureAndRelativeHumidity(i2c *i2c.I2C,
	precision MeasureRepeatability) (float32, float32, error) {

	ut, urh, err := v.ReadUncompTemperatureAndHumidity(i2c, precision)
	if err != nil {
		return 0, 0, err
	}
	lg.Debugf("Temperature and humidity uncompensated = %v, %v", ut, urh)
	temp := v.uncompTemperatureToCelsius(ut)
	rh := v.uncompHumidityToRelativeHumidity(urh)
	return temp, rh, nil
}

func (v *SHT3X) ReadTemperatureAndRelativeHumidityFarenheit(i2c *i2c.I2C,
	precision MeasureRepeatability) (float32, float32, error) {

	ut, urh, err := v.ReadUncompTemperatureAndHumidity(i2c, precision)
	if err != nil {
		return 0, 0, err
	}
	lg.Debugf("Temperature and humidity uncompensated = %v, %v", ut, urh)
	temp := v.uncompTemperatureToFarenheit(ut)
	rh := v.uncompHumidityToRelativeHumidity(urh)
	return temp, rh, nil
}

// Convert uncompensated humidity to relative humidity.
func (v *SHT3X) uncompHumidityToRelativeHumidity(uh uint16) float32 {
	rh := float32(uh) * 100 / (0x10000 - 1)
	rh2 := round32(rh, 2)
	return rh2
}

// Convert uncompensated temperature to Celsius value.
func (v *SHT3X) uncompTemperatureToCelsius(ut uint16) float32 {
	temp := float32(ut)*175/(0x10000-1) - 45
	temp2 := round32(temp, 2)
	return temp2
}

// Convert uncompensated temperature to Farenheit value.
func (v *SHT3X) uncompTemperatureToFarenheit(ut uint16) float32 {
	temp := float32(ut)*315/(0x10000-1) - 49
	temp2 := round32(temp, 2)
	return temp2
}


// Reset reboot a sensor.
func (v *SHT3X) Reset(i2c *i2c.I2C) error {
	lg.Debug("Reset sensor...")
	cmd := CMD_RESET
	_, err := i2c.WriteBytes(cmd)
	if err != nil {
		return err
	}
	v.lastCmd = cmd
	// Power-up time from specification
	time.Sleep(time.Microsecond * 1500)
	return nil
}
