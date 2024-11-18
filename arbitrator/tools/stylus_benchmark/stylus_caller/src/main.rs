use user_host::link::programs__create_stylus_config;

fn main() {
    println!("Hello, world!");

    unsafe {
        let ret = programs__create_stylus_config(0, 10000, 1, 1);
        println!("ret= {}", ret);
    }
}
