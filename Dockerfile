FROM maven:3-jdk-8-alpine

WORKDIR /usr/src/clustercode

ENV \
    CC_DEFAULT_DIR="/usr/src/clustercode/default" \
    CC_CONFIG_FILE="/usr/src/clustercode/config/clustercode.properties" \
    CC_CONFIG_DIR="/usr/src/clustercode/config" \
    CC_LOG_CONFIG_FILE="default/log4j2.xml"

VOLUME \
    /input \
    /output \
    $CC_CONFIG_DIR

EXPOSE \
    7600/tcp 7600/udp

CMD ["/usr/src/clustercode/docker-entrypoint.sh"]

COPY pom.xml docker ./
COPY src src/

RUN \
    mvn package -P package -e -B \
        -Dorg.slf4j.simpleLogger.log.org.apache.maven.cli.transfer.Slf4jMavenTransferListener=warn && \
    apk add --no-cache ffmpeg && \
    mv target/clustercode-jar-with-dependencies.jar clustercode.jar && \
    rm -r src && \
    rm -r target && \
    rm pom.xml