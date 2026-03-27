import { defineConfig } from "vocs";
import React from "react";

export default defineConfig({
  title: "ArbReth",
  description: "A modular, Rust-native execution client for Arbitrum",
  logoUrl: "/logo.png",
  iconUrl: "/favicon.png",
  topNav: [
    { text: "Docs", link: "/docs/introduction" },
    { text: "API Reference", link: "/docs/api-reference" },
    {
      element: React.createElement(
        "a",
        { href: "/rustdoc/arb_node/index.html", target: "_self", style: { fontSize: "inherit", color: "inherit", textDecoration: "none" } },
        "Rustdoc"
      ),
    },
    {
      text: "GitHub",
      link: "https://github.com/0xBloctopus/arbreth",
    },
  ],
  sidebar: [
    {
      text: "Overview",
      items: [
        { text: "Introduction", link: "/docs/introduction" },
        { text: "Architecture", link: "/docs/architecture" },
      ],
    },
    {
      text: "Running a Node",
      items: [
        { text: "Installation", link: "/docs/installation" },
        { text: "Configuration", link: "/docs/configuration" },
      ],
    },
    {
      text: "Technical Reference",
      items: [
        { text: "Transaction Types", link: "/docs/transaction-types" },
        { text: "JSON-RPC API", link: "/docs/json-rpc" },
        { text: "ArbOS State Machine", link: "/docs/arbos" },
        { text: "Precompiles", link: "/docs/precompiles" },
        { text: "Stylus WASM", link: "/docs/stylus" },
      ],
    },
    {
      text: "Development",
      items: [
        { text: "API Reference", link: "/docs/api-reference" },
        { text: "Contributing", link: "/docs/contributing" },
      ],
    },
  ],
  editLink: {
    pattern:
      "https://github.com/0xBloctopus/arbreth/edit/master/docs/docs/pages/:path",
    text: "Edit on GitHub",
  },
});
