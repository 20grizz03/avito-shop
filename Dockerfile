FROM ubuntu:latest
LABEL authors="maks"

ENTRYPOINT ["top", "-b"]