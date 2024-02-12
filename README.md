# hikariboxu

## Description

GO project which will encode and decode your data to and from video format.

Infinite storage glitch if you save it in free streaming services (Incoming).

This has dependencies with ffmpeg https://ffmpeg.org/

## Installation


```bash
git clone git@github.com:myan-ish/hikariboxu.git
cd hikariboxu

#Check env file for some configuration

#To encode
go run main.go <file_path> encode

#This will generate encoded_vid.mkv

#To decode
go run main.go <encoded_file> decode

#This will create file name decoded give the original extension to it and you can use it as expected
