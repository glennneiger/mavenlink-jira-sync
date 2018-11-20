package utility

import "log"

type FormatLogInterface interface {
	LevelZeroLog(marker string, text string)
	LevelOneLog(marker string, text string)
	LevelTwoLog(marker string, text string)
	LevelLog(level string, marker string, text string)
}
type FormatLog struct{}

func (fl *FormatLog) LevelZeroLog(marker string, text string) {
	log.Printf("%s %s", marker, text)
}
func (fl *FormatLog) LevelOneLog(marker string, text string) {
	log.Printf("%s%s %s", LevelOne, marker, text)
}
func (fl *FormatLog) LevelTwoLog(marker string, text string) {
	log.Printf("%s%s %s", LevelTwo, marker, text)
}
func (fl *FormatLog) LevelLog(level string, marker string, text string) {
	log.Printf("%s%s %s", level, marker, text)
}
