package main

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main()  {
	highPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool{
		return lev >= zap.ErrorLevel
	})

	lowPriority := zap.LevelEnablerFunc(func(lev zapcore.Level) bool {
		return lev < zap.ErrorLevel && lev >= zap.DebugLevel
	})
	zapcore.NewCore()
	zap.New(zapcore.NewTee())
}