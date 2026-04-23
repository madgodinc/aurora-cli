from setuptools import setup

setup(
    name="aurora-cli",
    version="0.5.0",
    description="CLI-клиент для Aurora AI",
    py_modules=["aurora"],
    python_requires=">=3.8",
    install_requires=["httpx"],
    entry_points={
        "console_scripts": [
            "aurora=aurora:main",
        ],
    },
)
