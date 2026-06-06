# Ktor + kotlinx.serialization need their reflection/metadata kept.
-keep class kotlinx.serialization.** { *; }
-keepclassmembers class * {
    @kotlinx.serialization.Serializable <fields>;
}
-keep,includedescriptorclasses class com.letzplay.musix.**$$serializer { *; }
-keepclassmembers class com.letzplay.musix.** {
    *** Companion;
}

# Ktor uses SLF4J; silence the missing logger backend.
-dontwarn org.slf4j.**
-dontwarn io.netty.**
-keep class io.ktor.** { *; }
-keep class kotlin.reflect.jvm.internal.** { *; }
