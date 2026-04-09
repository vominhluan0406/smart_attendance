"use client";

import { useState } from "react";
import { PlusCircle } from "lucide-react";

function base64urlToBuffer(base64url: string): Uint8Array {
  const padding = "==".slice(0, (4 - (base64url.length % 4)) % 4);
  const base64 = (base64url + padding).replace(/-/g, "+").replace(/_/g, "/");
  const raw = atob(base64);
  return Uint8Array.from(raw, (c) => c.charCodeAt(0));
}

function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let str = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    str += String.fromCharCode(bytes[i]);
  }
  return btoa(str).replace(/\+/g, "-").replace(/\//g, "_").replace(/=/g, "");
}

export default function WebAuthnButton() {
  const [status, setStatus] = useState<{
    type: "info" | "success" | "error";
    message: string;
  } | null>(null);
  const [loading, setLoading] = useState(false);

  const baseUrl = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

  async function handleRegister() {
    setLoading(true);
    setStatus({ type: "info", message: "Dang khoi tao yeu cau..." });

    try {
      // 1. Get options from server
      const beginResp = await fetch(`${baseUrl}/api/webauthn/register/begin`, {
        credentials: "include",
      });
      if (!beginResp.ok) throw new Error("Khong the bat dau dang ky");

      const options = await beginResp.json();

      // 2. Base64 decode fields as required by WebAuthn
      options.publicKey.challenge = base64urlToBuffer(
        options.publicKey.challenge
      );
      options.publicKey.user.id = base64urlToBuffer(options.publicKey.user.id);
      if (options.publicKey.excludeCredentials) {
        options.publicKey.excludeCredentials.forEach(
          (c: { id: string | Uint8Array }) => {
            c.id = base64urlToBuffer(c.id as string);
          }
        );
      }

      // 3. Create credentials using native browser API
      const credential = (await navigator.credentials.create({
        publicKey: options.publicKey,
      })) as PublicKeyCredential;

      if (!credential) throw new Error("Khong the tao credential");

      const attestationResponse =
        credential.response as AuthenticatorAttestationResponse;

      // 4. Encode result to send back to server
      const credJson = {
        id: credential.id,
        rawId: bufferToBase64url(credential.rawId),
        type: credential.type,
        response: {
          attestationObject: bufferToBase64url(
            attestationResponse.attestationObject
          ),
          clientDataJSON: bufferToBase64url(
            attestationResponse.clientDataJSON
          ),
        },
      };

      // 5. Send back to server
      const finishResp = await fetch(
        `${baseUrl}/api/webauthn/register/finish`,
        {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          credentials: "include",
          body: JSON.stringify(credJson),
        }
      );

      if (!finishResp.ok) {
        const err = await finishResp.json();
        throw new Error(err.error || "Dang ky that bai");
      }

      setStatus({
        type: "success",
        message: "Dang ky thiet bi thanh cong!",
      });
    } catch (err) {
      setStatus({
        type: "error",
        message: `Loi: ${err instanceof Error ? err.message : "Unknown error"}`,
      });
    } finally {
      setLoading(false);
    }
  }

  const statusStyles = {
    info: "bg-blue-50 text-blue-700",
    success: "bg-emerald-50 text-emerald-700",
    error: "bg-red-50 text-red-700",
  };

  return (
    <>
      {status && (
        <div
          className={`mb-4 p-3 rounded-xl text-sm ${
            statusStyles[status.type]
          }`}
        >
          {status.message}
        </div>
      )}

      <button
        onClick={handleRegister}
        disabled={loading}
        className="w-full flex items-center justify-center gap-2 bg-emerald-600 text-white px-6 py-3 rounded-xl font-bold shadow-lg hover:bg-emerald-700 active:scale-95 transition-all disabled:opacity-50"
      >
        {loading ? (
          <span className="spinner" />
        ) : (
          <PlusCircle className="w-5 h-5" />
        )}
        Dang ky thiet bi moi
      </button>
    </>
  );
}
