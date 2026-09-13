#!/bin/bash

case $1 in
    start)
        docker run --name=%NAME% \
        --net=domoticanet \
        --restart=unless-stopped \
        -d \
        -e TZ=Europe/Amsterdam \
        -e INFLUXDB_URL="http://influxdb:8086" \
        -e P1_SERIAL_PATH="/dev/ttyUSB0" \
        --device /dev/serial/by-id/usb-FTDI_FT232R_USB_UART_AK43OI8Y-if00-port0:/dev/ttyUSB0 \
        %NAME%
        ;;
    stop)
        docker stop %NAME% | xargs docker rm
        ;;
    restart)
        $0 stop
        $0 start
        ;;
    logs)
        docker logs %NAME%
        ;;
    shell)
        docker exec -ti %NAME% /bin/sh
        ;;
    *)
        echo "unknown or missing parameter $1"
esac
