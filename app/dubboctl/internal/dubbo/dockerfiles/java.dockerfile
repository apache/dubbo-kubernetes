# Please modify your template according to your business needs!!!

FROM openjdk:8-jdk-alpine

ADD target/demo-0.0.1-SNAPSHOT.jar app.jar
ENV JAVA_OPTS=""
ENTRYPOINT exec java $JAVA_OPTS -jar /app.jar
