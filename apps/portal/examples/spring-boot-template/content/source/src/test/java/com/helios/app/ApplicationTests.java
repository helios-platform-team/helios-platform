package com.helios.app;

import org.junit.jupiter.api.Test;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.test.context.ActiveProfiles;

/**
 * Smoke test ensuring the Spring application context loads successfully.
 *
 * <p>Uses an in-memory H2 database (via the {@code test} profile) so
 * the test suite runs without an external PostgreSQL instance — critical
 * for CI pipelines (Tekton) where a DB may not yet be provisioned.</p>
 */
@SpringBootTest
@ActiveProfiles("test")
class ApplicationTests {

    @Test
    void contextLoads() {
        // If the context fails to load, this test will fail.
    }
}
