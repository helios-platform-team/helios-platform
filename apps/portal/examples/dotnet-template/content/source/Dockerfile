FROM mcr.microsoft.com/dotnet/sdk:10.0.201 AS build
WORKDIR /src

COPY DotNetApi.csproj ./
RUN dotnet restore DotNetApi.csproj

COPY . .
RUN dotnet publish DotNetApi.csproj -c Release -o /app/publish /p:UseAppHost=false

FROM mcr.microsoft.com/dotnet/aspnet:10.0.5 AS runtime
WORKDIR /app
COPY --from=build /app/publish .

ENV ASPNETCORE_URLS=http://+:${{ values.port }}
EXPOSE ${{ values.port }}

ENTRYPOINT ["dotnet", "DotNetApi.dll"]
