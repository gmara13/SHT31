package main

import (
	i2c "github.com/d2r2/go-i2c"
	logger "github.com/d2r2/go-logger"
	sht3x "github.com/gmara13/SHT31"
)

var lg = logger.NewPackageLogger("main",
	logger.DebugLevel,
	// logger.InfoLevel,
)

func main() {
	defer logger.FinalizeLogger()
	// Create new connection to i2c-bus on 0 line with address 0x44.
	// Use i2cdetect utility to find device address over the i2c-bus
	i2c, err := i2c.NewI2C(0x44, 1)
	if err != nil {
		lg.Fatal(err)
	}
	defer i2c.Close()

	// Uncomment/comment next line to suppress/increase verbosity of output
	// logger.ChangePackageLogLevel("i2c", logger.InfoLevel)
	// logger.ChangePackageLogLevel("sht3x", logger.InfoLevel)

	sensor := sht3x.NewSHT3X()
	// Clear sensor settings
	err = sensor.Reset(i2c)
	if err != nil {
		lg.Fatal(err)
	}


	lg.Notify("**********************************************************************************************")
	lg.Notify("*** Single shot measurement mode")
	lg.Notify("**********************************************************************************************")
	temp, rh, err := sensor.ReadTemperatureAndRelativeHumidity(i2c, sht3x.RepeatabilityLow)
	if err != nil {
		lg.Fatal(err)
	}
	lg.Infof("Temperature and relative humidity = %v*C, %v%%", temp, rh)

	temp, rh, err = sensor.ReadTemperatureAndRelativeHumidityFarenheit(i2c, sht3x.RepeatabilityHigh)
	if err != nil {
		lg.Fatal(err)
	}
	lg.Infof("Temperature and relative humidity = %v*F, %v%%", temp, rh)

}
