FROM openjdk:{{.Version}}-jdk

ADD {{.Jar}} /{{.ExeName}}.jar
{{if .Port}}
EXPOSE {{.}}
{{end}}
ENTRYPOINT exec java -jar /{{.ExeName}}.jar
