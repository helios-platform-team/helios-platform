var builder = WebApplication.CreateBuilder(args);

builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

{% if values.hasDatabase -%}
var dbHost = Environment.GetEnvironmentVariable("DB_HOST") ?? "localhost";
var dbUser = Environment.GetEnvironmentVariable("DB_USER") ?? "postgres";
var dbPassword = Environment.GetEnvironmentVariable("DB_PASSWORD")
    ?? Environment.GetEnvironmentVariable("DB_PASS")
    ?? "postgres";
var dbName = Environment.GetEnvironmentVariable("DB_NAME") ?? "${{ values.name }}-db";

var connectionTemplate = builder.Configuration.GetConnectionString("DefaultConnection")
    ?? "Host={DB_HOST};Username={DB_USER};Password={DB_PASSWORD};Database={DB_NAME}";
var connectionString = connectionTemplate
    .Replace("{DB_HOST}", dbHost, StringComparison.Ordinal)
    .Replace("{DB_USER}", dbUser, StringComparison.Ordinal)
    .Replace("{DB_PASSWORD}", dbPassword, StringComparison.Ordinal)
    .Replace("{DB_NAME}", dbName, StringComparison.Ordinal);
builder.Configuration["ConnectionStrings:DefaultConnection"] = connectionString;

// Register health checks
builder.Services.AddHealthChecks()
    .AddNpgSql(connectionString);
{% endif -%}
builder.Services.AddCors();


var app = builder.Build();

app.UseCors(policy => policy.AllowAnyOrigin().AllowAnyHeader().AllowAnyMethod());

if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();

{% if values.hasDatabase -%}
    app.MapGet("/database/config", () =>
        Results.Ok(new
        {
            Host = dbHost,
            User = dbUser,
            Database = dbName,
        }));
{% endif -%}
}

app.UseHttpsRedirection();
app.MapControllers();

{% if values.hasDatabase -%}
app.MapHealthChecks("/health", new Microsoft.AspNetCore.Diagnostics.HealthChecks.HealthCheckOptions
{
    ResponseWriter = async (context, report) =>
    {
        context.Response.ContentType = "application/json";
        await context.Response.WriteAsJsonAsync(new { status = report.Status.ToString() });
    }
});
{% else -%}
app.MapGet("/health", () => Results.Ok(new { status = "ok" }));
{% endif -%}

app.Run();
