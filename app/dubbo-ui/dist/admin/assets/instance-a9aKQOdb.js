import{r as n}from"./request-fr6f-AQz.js";const s=t=>n({url:"/instance/search",method:"get",params:t}),c=t=>n({url:"/instance/detail",method:"get",params:t}),o=t=>n({url:"/instance/metrics",method:"get",params:t}),i=(t,e)=>n({url:"/instance/config/operatorLog",method:"get",params:{instanceIP:t,appName:e}}),u=(t,e,a)=>n({url:"/instance/config/operatorLog",method:"put",params:{instanceIP:t,appName:e,operatorLog:a}}),g=(t,e)=>n({url:"/instance/config/trafficDisable",method:"get",params:{instanceIP:t,appName:e}}),f=(t,e,a)=>n({url:"/instance/config/trafficDisable",method:"put",params:{instanceIP:t,appName:e,trafficDisable:a}});export{o as a,f as b,i as c,g as d,c as g,s,u};