package com.helios.app.controller;

import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RestController;

import java.time.Instant;
import java.util.Map;

/**
 * Root controller providing basic health-check and greeting endpoints.
 *
 * <p>These endpoints serve two purposes:
 * <ol>
 *   <li>Give developers immediate visual feedback that the app is running.</li>
 *   <li>Provide a lightweight HTTP check independent of Spring Actuator
 *       for quick smoke-testing during CI/CD.</li>
 * </ol>
 */
@RestController
public class AppController {

    @GetMapping("/")
    public ResponseEntity<Map<String, String>> hello() {
        return ResponseEntity.ok(Map.of(
                "message", "Hello from ${{ values.name }}!",
                "timestamp", Instant.now().toString()
        ));
    }

    @GetMapping("/health")
    public ResponseEntity<Map<String, String>> health() {
        return ResponseEntity.ok(Map.of("status", "ok"));
    }
}
