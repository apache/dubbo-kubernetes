FROM openjdk:8
 
COPY target/lib /app/lib
COPY target/kubernetes-demo-provider-1.0.0-SNAPSHOT.jar /app/kubernetes-demo-provider-1.0.0-SNAPSHOT.jar

EXPOSE 8080
ENTRYPOINT ["java", "-jar", "/app/kubernetes-demo-provider-1.0.0-SNAPSHOT.jar"]
 

