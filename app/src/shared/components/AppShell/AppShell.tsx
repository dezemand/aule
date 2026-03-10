import { useState } from "react";
import { Link, useRouterState } from "@tanstack/react-router";
import {
  AppShell as MantineAppShell,
  Burger,
  Stack,
  Text,
  Tooltip,
  UnstyledButton,
  Title,
} from "@mantine/core";
import { IconHome2, IconChecklist, IconCpu } from "@tabler/icons-react";
import type { Icon } from "@tabler/icons-react";
import type { ReactNode } from "react";
import { TopBar } from "@/shared/components/TopBar/TopBar";
import classes from "./AppShell.module.css";

type SubPage = {
  to: string;
  label: string;
};

type NavSection = {
  id: string;
  label: string;
  icon: Icon;
  basePath: string;
  subPages: SubPage[];
};

const NAV_SECTIONS: NavSection[] = [
  {
    id: "dashboard",
    label: "Dashboard",
    icon: IconHome2,
    basePath: "/",
    subPages: [{ to: "/", label: "Overview" }],
  },
  {
    id: "tasks",
    label: "Tasks",
    icon: IconChecklist,
    basePath: "/tasks",
    subPages: [{ to: "/tasks", label: "All Tasks" }],
  },
  {
    id: "agents",
    label: "Agent Types",
    icon: IconCpu,
    basePath: "/agent-types",
    subPages: [{ to: "/agent-types", label: "All Types" }],
  },
];

function getActiveSection(pathname: string): NavSection {
  for (const section of NAV_SECTIONS) {
    if (section.basePath === "/") {
      if (pathname === "/") return section;
    } else if (pathname.startsWith(section.basePath)) {
      return section;
    }
  }
  return NAV_SECTIONS[0]!;
}

export function AppShell({ children }: { children: ReactNode }) {
  const routerState = useRouterState();
  const currentPath = routerState.location.pathname;
  const [opened, setOpened] = useState(false);
  const activeSection = getActiveSection(currentPath);

  const sectionIcons = NAV_SECTIONS.map((section) => (
    <Tooltip
      label={section.label}
      position="right"
      withArrow
      transitionProps={{ duration: 0 }}
      key={section.id}
    >
      <UnstyledButton
        component={Link}
        to={section.basePath}
        onClick={() => setOpened(false)}
        className={classes.mainLink}
        data-active={section.id === activeSection.id || undefined}
        aria-label={section.label}
      >
        <section.icon size={22} stroke={1.5} />
      </UnstyledButton>
    </Tooltip>
  ));

  const subPageLinks = activeSection.subPages.map((page) => (
    <Link
      className={classes.link}
      data-active={currentPath === page.to || undefined}
      to={page.to}
      key={page.to}
      onClick={() => setOpened(false)}
    >
      {page.label}
    </Link>
  ));

  return (
    <MantineAppShell
      navbar={{
        width: 300,
        breakpoint: "sm",
        collapsed: { mobile: !opened },
      }}
      padding={0}
    >
      <MantineAppShell.Navbar p={0}>
        <nav className={classes.navbar}>
          <div className={classes.wrapper}>
            <div className={classes.aside}>
              <div className={classes.logo}>
                <Text fw={700} size="lg">
                  A
                </Text>
              </div>
              {sectionIcons}
              <div className={classes.burgerContainer}>
                <Burger
                  opened={opened}
                  onClick={() => setOpened((v) => !v)}
                  hiddenFrom="sm"
                  size="sm"
                  aria-label="Toggle navigation"
                />
              </div>
            </div>

            <div className={classes.main}>
              <Title order={4} className={classes.title}>
                {activeSection.label}
              </Title>
              <Stack gap={0} className={classes.links}>
                {subPageLinks}
              </Stack>
            </div>
          </div>
        </nav>
      </MantineAppShell.Navbar>

      <MantineAppShell.Main className={classes.mainContent}>
        <TopBar />
        <div className={classes.page}>{children}</div>
      </MantineAppShell.Main>
    </MantineAppShell>
  );
}
