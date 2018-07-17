FROM openjdk:8
 
COPY target/lib /app/lib
COPY target/kubernetes-demo-consumer-1.0.0-SNAPSHOT.jar /app/kubernetes-demo-consumer-1.0.0-SNAPSHOT.jar

EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app/kubernetes-demo-consumer-1.0.0-SNAPSHOT.jar"]
 

