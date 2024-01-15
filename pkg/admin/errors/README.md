# Project Error Tips

## Controller layer
- Use c.Error() to append a BizError to the gin.Context, and the ErrorHandler middleware will print log and return the unified error response.
## Service layer
- When an error occurs for the first time, use NewBizError or NewBizErrorWithStack to wrap the original error.
- If a service method is called by another service method, the returned error should be returned directly, only wrap error at the first time.
- If a service method calls an external dependency and return an error, we should wrap the error with NewBizErrorWithStack, and the invoke stack will be printed in the log.