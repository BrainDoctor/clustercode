package clustercode.main;

import clustercode.api.config.ConfigLoader;
import clustercode.main.modules.*;
import com.google.inject.Guice;
import com.google.inject.Injector;
import com.google.inject.Module;
import lombok.extern.slf4j.XSlf4j;

import java.io.IOException;
import java.io.InputStream;
import java.util.LinkedList;
import java.util.List;
import java.util.jar.Manifest;

@XSlf4j
public class GuiceManager {

    private final Injector injector;

    public GuiceManager(ConfigLoader loader) {
        log.debug("Creating guice modules...");
        List<Module> modules = new LinkedList<>();

        modules.add(new GlobalModule());
        modules.add(new CleanupModule(loader));
        modules.add(new ClusterModule(loader));
        modules.add(new ConstraintModule(loader));
        modules.add(new ProcessModule());
        modules.add(new ScanModule(loader));
        modules.add(new TranscodeModule(loader));
        modules.add(new RestApiModule(loader));

        log.info("Booting clustercode {}...", getApplicationVersion());
        injector = Guice.createInjector(modules);
    }

    public <T> T getInstance(Class<T> clazz) {
        return injector.getInstance(clazz);
    }

    private static String getApplicationVersion() {
        InputStream stream = ClassLoader.getSystemResourceAsStream("META-INF/MANIFEST.MF");
        try {
            return new Manifest(stream).getMainAttributes().getValue("Implementation-Version");
        } catch (IOException | NullPointerException e) {
            log.catching(e);
        }
        return "unknown-version";
    }


}