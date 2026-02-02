import org.gradle.testing.jacoco.plugins.JacocoTaskExtension
import org.jetbrains.intellij.platform.gradle.TestFrameworkType

plugins {
    id("java")
    id("org.jetbrains.kotlin.jvm") version "2.0.21"
    id("org.jetbrains.intellij.platform") version "2.11.0"
    id("org.jlleitschuh.gradle.ktlint") version "13.0.0"
    id("io.gitlab.arturbosch.detekt") version "1.23.8"
    jacoco
}

group = providers.gradleProperty("pluginGroup").get()
version = providers.gradleProperty("pluginVersion").get()

repositories {
    mavenCentral()
    intellijPlatform {
        defaultRepositories()
    }
}

kotlin {
    jvmToolchain(22)
    compilerOptions {
        jvmTarget.set(org.jetbrains.kotlin.gradle.dsl.JvmTarget.JVM_21)
    }
}

java {
    toolchain {
        languageVersion.set(JavaLanguageVersion.of(22))
    }
    sourceCompatibility = JavaVersion.VERSION_21
    targetCompatibility = JavaVersion.VERSION_21
}

dependencies {
    intellijPlatform {
        val platformType = providers.gradleProperty("platformType")
        val platformVersion = providers.gradleProperty("platformVersion")
        create(platformType, platformVersion)

        bundledPlugin("com.intellij.java")
        pluginVerifier()
        zipSigner()
        testFramework(TestFrameworkType.Platform)
    }

    // HTTP client
    implementation("com.squareup.okhttp3:okhttp:4.12.0")
    implementation("com.squareup.okhttp3:okhttp-sse:4.12.0")

    // JSON serialization
    implementation("com.google.code.gson:gson:2.11.0")

    // Coroutines - provided by IntelliJ Platform, no need to add explicitly

    // Testing
    testImplementation("org.junit.jupiter:junit-jupiter:5.11.4")
    testRuntimeOnly("org.junit.platform:junit-platform-launcher")
    testImplementation("io.mockk:mockk:1.13.14")
    testImplementation("com.squareup.okhttp3:mockwebserver:4.12.0")
}

intellijPlatform {
    pluginConfiguration {
        name = providers.gradleProperty("pluginName")
        version = providers.gradleProperty("pluginVersion")

        ideaVersion {
            sinceBuild = providers.gradleProperty("pluginSinceBuild")
            val untilBuildValue = providers.gradleProperty("pluginUntilBuild").orNull
            if (!untilBuildValue.isNullOrBlank()) {
                untilBuild.set(untilBuildValue)
            }
        }

        vendor {
            name = "Valksor"
            url = "https://valksor.com"
            email = "support@valksor.com"
        }
    }

    signing {
        certificateChain = providers.environmentVariable("CERTIFICATE_CHAIN")
        privateKey = providers.environmentVariable("PRIVATE_KEY")
        password = providers.environmentVariable("PRIVATE_KEY_PASSWORD")
    }

    publishing {
        token = providers.environmentVariable("PUBLISH_TOKEN")
        channels = listOf(providers.gradleProperty("publishChannel").getOrElse("default"))
    }

    pluginVerification {
        ides {
            recommended()
        }
    }
}

// ktlint configuration
ktlint {
    version.set("1.5.0")
    android.set(false)
    ignoreFailures.set(false)
    reporters {
        reporter(org.jlleitschuh.gradle.ktlint.reporter.ReporterType.PLAIN)
        reporter(org.jlleitschuh.gradle.ktlint.reporter.ReporterType.SARIF)
    }
    filter {
        exclude("**/generated/**")
    }
}

// detekt configuration
detekt {
    config.setFrom("$projectDir/config/detekt/detekt.yml")
    buildUponDefaultConfig = true
    allRules = false
    parallel = true
    autoCorrect = false
    ignoreFailures = false
}

tasks {
    wrapper {
        gradleVersion = "9.3.1"
    }

    register("quality") {
        group = "verification"
        description = "Run all quality checks (ktlint + detekt)"
        dependsOn("ktlintCheck", "detekt")
    }

    register("formatKotlin") {
        group = "formatting"
        description = "Format Kotlin files with ktlint"
        dependsOn("ktlintFormat")
    }

    test {
        useJUnitPlatform()
        finalizedBy(jacocoTestReport)
        // Fix for JaCoCo 0% coverage with IntelliJ Platform plugin
        // See: https://github.com/JetBrains/intellij-platform-gradle-plugin/issues/1383
        configure<JacocoTaskExtension> {
            isIncludeNoLocationClasses = true
            excludes = listOf("jdk.internal.*")
        }
    }

    jacocoTestReport {
        dependsOn(test)
        reports {
            xml.required.set(true)
            html.required.set(true)
        }
        // Use instrumented classes to match runtime execution data
        classDirectories.setFrom(named("instrumentCode"))
    }

    buildSearchableOptions {
        enabled = false
    }
}
