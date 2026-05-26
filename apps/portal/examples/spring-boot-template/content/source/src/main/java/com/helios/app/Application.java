package com.helios.app;

import org.springframework.boot.SpringApplication;
import org.springframework.boot.autoconfigure.SpringBootApplication;

/**
 * Entry point for the ${{ values.name }} Spring Boot application.
 *
 * <p>The {@code @SpringBootApplication} annotation enables:
 * <ul>
 *   <li>Component scanning within the {@code com.helios.app} package</li>
 *   <li>Auto-configuration of Spring Data JPA, Actuator, etc.</li>
 *   <li>Property source resolution from {@code application.yml}</li>
 * </ul>
 */
@SpringBootApplication
public class Application {

    public static void main(String[] args) {
        SpringApplication.run(Application.class, args);
    }
}
