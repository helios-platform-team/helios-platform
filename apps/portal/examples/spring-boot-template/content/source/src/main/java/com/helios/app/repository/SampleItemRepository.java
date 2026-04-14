package com.helios.app.repository;

import com.helios.app.entity.SampleItem;
import org.springframework.data.jpa.repository.JpaRepository;
import org.springframework.stereotype.Repository;

/**
 * Spring Data JPA repository for {@link SampleItem}.
 *
 * <p>Provides CRUD operations and query derivation. Developers can add
 * custom query methods following Spring Data naming conventions.</p>
 */
@Repository
public interface SampleItemRepository extends JpaRepository<SampleItem, Long> {
}
