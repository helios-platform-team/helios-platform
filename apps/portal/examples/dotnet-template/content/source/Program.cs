using Microsoft.AspNetCore.Mvc;

var builder = WebApplication.CreateBuilder(args);

builder.Services.AddControllers();
builder.Services.AddEndpointsApiExplorer();
builder.Services.AddSwaggerGen();

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

var app = builder.Build();

if (app.Environment.IsDevelopment())
{
    app.UseSwagger();
    app.UseSwaggerUI();
}

app.UseHttpsRedirection();
app.MapControllers();

app.MapGet("/health", () => Results.Ok(new { status = "ok" }));
app.MapGet("/database/config", ([FromServices] IConfiguration config) =>
    Results.Ok(new
    {
        Host = dbHost,
        User = dbUser,
        Database = dbName,
        ConnectionString = config.GetConnectionString("DefaultConnection")
    }));

app.Run();
