import{r as o}from"./request-fr6f-AQz.js";const a=async e=>o({url:"promQL/query",method:"get",params:e});async function c(e){var t,u;try{let r=(u=(t=await a({query:e}))==null?void 0:t.data)==null?void 0:u.result[0];return r!=null&&r.value&&r.value.length>0?Number(r.value[1]):"NA"}catch(r){console.error("fetch from prom error: ",r)}return"NA"}export{c as q};