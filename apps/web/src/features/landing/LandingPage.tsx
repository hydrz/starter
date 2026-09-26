import { ArchitectureSection } from "./ArchitectureSection";
import { CtaSection } from "./CtaSection";
import { FeaturesSection } from "./FeaturesSection";
import { HeroSection } from "./HeroSection";
import { LandingFooter } from "./LandingFooter";
import { LandingNavbar } from "./LandingNavbar";
import { TechStackSection } from "./TechStackSection";

export function LandingPage() {
  return (
    <div className="min-h-screen bg-background text-foreground flex flex-col">
      <LandingNavbar />
      <main className="flex-1">
        <HeroSection />
        <FeaturesSection />
        <ArchitectureSection />
        <TechStackSection />
        <CtaSection />
      </main>
      <LandingFooter />
    </div>
  );
}
