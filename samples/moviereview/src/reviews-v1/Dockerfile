FROM eclipse-temurin:21-jdk-alpine AS build

WORKDIR /src
COPY ReviewsApplication.java .
RUN javac ReviewsApplication.java

FROM eclipse-temurin:21-jre-alpine
WORKDIR /app
COPY --from=build /src/*.class .

EXPOSE 9080
CMD ["java", "-cp", "/app", "ReviewsApplication"]
