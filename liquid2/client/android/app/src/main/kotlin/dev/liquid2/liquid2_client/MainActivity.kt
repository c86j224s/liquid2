package dev.liquid2.liquid2_client

import android.content.Intent
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import java.util.UUID

class MainActivity : FlutterActivity() {
    private var channel: MethodChannel? = null
    private val pending = java.util.ArrayDeque<Map<String, String>>()
    override fun configureFlutterEngine(engine: FlutterEngine) {
        super.configureFlutterEngine(engine)
        channel = MethodChannel(engine.dartExecutor.binaryMessenger, "liquid2/share")
        capture(intent)
        channel!!.setMethodCallHandler { call, result ->
            when (call.method) {
                "takeShare" -> { result.success(pending.pollFirst()) }
                "getServer" -> result.success(getPreferences(0).getString("server", ""))
                "setServer" -> { getPreferences(0).edit().putString("server", call.arguments as String).apply(); result.success(null) }
                else -> result.notImplemented()
            }
        }
    }
    override fun onNewIntent(intent: Intent) {
        super.onNewIntent(intent)
        setIntent(intent)
        capture(intent)
        channel?.invokeMethod("shareAvailable", null)
    }
    private fun capture(value: Intent?) {
        if (value?.action != Intent.ACTION_SEND || value.type != "text/plain") return
        val text = value.getStringExtra(Intent.EXTRA_TEXT) ?: return
        pending.addLast(mapOf("id" to UUID.randomUUID().toString(), "text" to text))
        value.action = Intent.ACTION_MAIN
        value.removeExtra(Intent.EXTRA_TEXT)
    }
}
